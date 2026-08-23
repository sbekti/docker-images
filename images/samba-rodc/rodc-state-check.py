#!/usr/bin/python3
"""Refuse Samba directory state that is missing, invalid, or writable."""

import sys
from typing import NoReturn

import ldb
from samba.auth import system_session
from samba.param import LoadParm
from samba.samdb import SamDB


CONFIG_PATH = "/etc/samba/smb.conf"


def fail(message: str) -> NoReturn:
    print(f"ERROR: {message}", file=sys.stderr)
    raise SystemExit(1)


def main() -> None:
    try:
        parameters = LoadParm()
        parameters.load(CONFIG_PATH)
        database_path = f"{parameters.get('private dir')}/sam.ldb"
        database = SamDB(
            url=f"tdb://{database_path}",
            session_info=system_session(),
            lp=parameters,
            flags=ldb.FLG_DONT_CREATE_DB,
        )
        is_rodc = database.am_rodc()
    except Exception as error:
        fail(f"Cannot validate Samba directory state: {error}")

    if not is_rodc:
        fail("Samba directory state is writable; refusing to start.")

    print("Samba directory state is a valid RODC.")


if __name__ == "__main__":
    main()
