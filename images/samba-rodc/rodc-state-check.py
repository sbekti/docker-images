#!/usr/bin/python3
"""Refuse missing, invalid, or writable Samba directory state."""

import os
import sys
from typing import NoReturn

import ldb
from samba.auth import system_session
from samba.param import LoadParm
from samba.samdb import SamDB


CONFIG_PATH = "/etc/samba/smb.conf"
EXPECTED_PRIVATE_DIR = "/var/lib/samba/private"
EXPECTED_SERVICES = ["rpc", "ldap", "cldap", "drepl", "winbindd", "kcc"]


def fail(message: str) -> NoReturn:
    print(f"ERROR: {message}", file=sys.stderr)
    raise SystemExit(1)


def required_environment(name: str) -> str:
    value = os.environ.get(name)
    if not value:
        fail(f"{name} is required.")
    return value


def require_parameter(load_parameters: LoadParm, name: str, expected: object) -> None:
    actual = load_parameters.get(name)
    if actual != expected:
        fail(f"Samba parameter {name!r} must be {expected!r}, not {actual!r}.")


def main() -> None:
    if not os.path.isfile(CONFIG_PATH):
        fail(f"Missing {CONFIG_PATH}.")

    load_parameters = LoadParm()
    try:
        load_parameters.load(CONFIG_PATH)
    except Exception as error:
        fail(f"Cannot load Samba configuration: {error}")

    configured_private_dir = load_parameters.get("private dir")
    if not configured_private_dir:
        fail("Samba configuration does not define a private dir.")

    private_dir = os.path.realpath(configured_private_dir)
    if private_dir != EXPECTED_PRIVATE_DIR:
        fail(
            "Samba private dir must be "
            f"{EXPECTED_PRIVATE_DIR}, not {private_dir}."
        )

    database_path = os.path.join(private_dir, "sam.ldb")
    if not os.path.isfile(database_path):
        fail(f"Missing {database_path}.")

    try:
        sam_database = SamDB(
            url=f"tdb://{database_path}",
            session_info=system_session(),
            lp=load_parameters,
            flags=ldb.FLG_DONT_CREATE_DB,
        )
        is_rodc = sam_database.am_rodc()
    except Exception as error:
        fail(f"Cannot validate existing Samba directory state: {error}")

    if not is_rodc:
        fail("Existing Samba directory state is writable; refusing to start.")

    realm = required_environment("REALM")
    domain = required_environment("DOMAIN")
    netbios_name = required_environment("NETBIOS_NAME")
    dns_hostname = required_environment("DNS_HOSTNAME")
    rodc_address = required_environment("RODC_ADDRESS")
    rpc_port_start = required_environment("RPC_PORT_START")
    rpc_port_end = required_environment("RPC_PORT_END")

    require_parameter(load_parameters, "realm", realm)
    require_parameter(load_parameters, "workgroup", domain)
    require_parameter(load_parameters, "netbios name", netbios_name)
    require_parameter(load_parameters, "dns hostname", dns_hostname)
    require_parameter(load_parameters, "interfaces", ["lo", rodc_address])
    require_parameter(load_parameters, "bind interfaces only", True)
    require_parameter(load_parameters, "server services", EXPECTED_SERVICES)
    require_parameter(
        load_parameters,
        "rpc server dynamic port range",
        f"{rpc_port_start}-{rpc_port_end}",
    )
    require_parameter(
        load_parameters,
        "ntlm auth",
        "mschapv2-and-ntlmv2-only",
    )
    require_parameter(load_parameters, "dns update command", ["/usr/bin/false"])
    require_parameter(load_parameters, "ldap server require strong auth", "Yes")

    print("Samba directory state and configuration are a valid RODC.")


if __name__ == "__main__":
    main()
