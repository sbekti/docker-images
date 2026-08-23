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


def fail(message: str) -> NoReturn:
    print(f"ERROR: {message}", file=sys.stderr)
    raise SystemExit(1)


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

    print("Samba directory state is a valid RODC.")


if __name__ == "__main__":
    main()
