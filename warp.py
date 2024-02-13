#!/usr/bin/env python3

"""
   Warp! - Your lazy ssh command line helper
"""
__version__ = "0.1.4"
__author__ = "P S, Adithya (adithya3494@gmail.com)"
__license__ = "MIT"

import os
import sys
import sqlite3
import argparse
from pathlib import Path
from iterfzf import iterfzf

HEADER = "\033[95m"
OKBLUE = "\033[94m"
OKCYAN = "\033[96m"
OKGREEN = "\033[92m"
WARNING = "\033[93m"
FAIL = "\033[91m"
ENDC = "\033[0m"
BOLD = "\033[1m"


def parse_args():
    parser = argparse.ArgumentParser()
    p_group = parser.add_mutually_exclusive_group()
    p_group.add_argument("-a", "--add", help="add connection(s)", action="store_true")
    p_group.add_argument(
        "-c",
        "--connect",
        help="initiate a connection from stored values",
        action="store_true",
    )
    p_group.add_argument(
        "-d", "--delete", help="delete connection(s)", action="store_true"
    )
    p_group.add_argument("-s", "--show", help="show all data", action="store_true")
    p_group.add_argument(
        "-o", "--output", help="write out existing data to a file", action="store_true"
    )
    return parser.parse_args()


def initialize_db():
    config_dir = Path.home() / ".config/warp"
    config_dir.mkdir(parents=True, exist_ok=True)
    db_file = config_dir / "warp.db"
    _connect = sqlite3.connect(db_file)
    cursor = _connect.cursor()
    cursor.execute(
        """CREATE TABLE IF NOT EXISTS main(environment text,hostname text,ip_address real,
                      username text,password text)"""
    )
    cursor.execute(
        """CREATE TABLE IF NOT EXISTS alias(name text,ip_address real,username text,password text)"""
    )
    return _connect, cursor


def pretty_print(data, header, separator):
    widths = [max(len(str(cell)), len(header[i])) for i, cell in enumerate(header)]
    formatted_row = " ".join("{:%d}" % width for width in widths)
    print("\t" + formatted_row.format(*header))
    print("\t" + formatted_row.format(*separator))
    for row in data:
        print("\t" + formatted_row.format(*row))


def add(cursor):
    header = column_header(cursor)
    separator = ["-" * len(h) for h in header]
    print(f"{BOLD} > Do you have a file to load ?{ENDC}")
    dec = input(f"{BOLD} > [Y/N] :{ENDC} ") or "n"
    if dec.upper() == "N":
        print(f" > Enter the details in '{BOLD}{','.join(header)}{ENDC}' format")
        a_inpt = [input(" > INPUT: ").split(",")]
    elif dec.upper() == "Y":
        print(f"{BOLD} > /path/to/file ?{ENDC}")
        try:
            with open(input(), "r", encoding="utf-8") as file:
                a_inpt = [
                    line.strip("\n").split(",")
                    for line in file
                    if not line.startswith("#")
                ]
        except FileNotFoundError:
            print(f"{BOLD} > {FAIL}file not found{ENDC}")
            sys.exit(1)
    else:
        terminate()
    data = [tuple(d) for d in a_inpt]
    print(f"{BOLD} > The below info will be added, proceed ?{ENDC}\n")
    pretty_print(data, header, separator)
    dec = input(f"{BOLD}\n > [Y/N] :{ENDC} ")
    if dec.upper() == "Y":
        insert_func(cursor, data)
    else:
        terminate()


def insert_func(cursor, data):
    cursor.executemany("INSERT INTO main VALUES (?,?,?,?,?)", data)
    print(f"{BOLD} > '{len(data)}' insert(s) {OKGREEN}successful{ENDC}")


def column_header(cursor):
    return [
        description[0]
        for description in cursor.execute("SELECT * FROM main").description
    ]


def connect(cursor):
    try:
        cursor.execute("SELECT * from main")
        conn_data = fzf_prompt(cursor.fetchall())
        ssh_command = f"ssh {conn_data[3]}@{conn_data[2]}"
        print(f"{BOLD} > Executing {OKCYAN}'{ssh_command}'{ENDC}")
        os.system(ssh_command)
    except KeyboardInterrupt:
        terminate()


def fzf_prompt(data):
    try:
        return iterfzf([" ".join(d) for d in data]).replace(" ", ",").split(",")
    except Exception as e:
        print(e)
        terminate()


def show(cursor):
    header = column_header(cursor)
    header.insert(0, "ID")
    separator = ["-" * len(h) for h in header]
    cursor.execute("SELECT rowid, * from main")
    data = cursor.fetchall()
    pretty_print(data, header, separator)


def delete(cursor):
    header = column_header(cursor)
    header.insert(0, "ID")
    separator = ["-" * len(h) for h in header]
    print(
        f" > Enter the rowid's to drop in {BOLD}id1,id2 or range:'id3_id6'{ENDC} format"
    )
    d_inpt = input(" > INPUT: ").split(",")
    rows = []
    try:
        for i in d_inpt:
            if "_" in i:
                start, end = map(int, i.split("_"))
                if start < end:
                    rows.extend(map(str, range(start, end + 1)))
                else:
                    rows.extend(map(str, range(start, end - 1, -1)))
            else:
                rows.append(i)
        data = []
        for row_id in rows:
            cursor.execute("""SELECT rowid, * from main WHERE rowid = ?""", (row_id,))
            data.append(cursor.fetchall()[0])
    except IndexError:
        print(f"\n{BOLD} > {WARNING}Invalid range, use '-s' to validate{ENDC}")
        sys.exit(1)
    print(f"{BOLD} > Below data would be deleted... proceed ?{ENDC}\n")
    pretty_print(data, header, separator)
    dec = input(f"{BOLD}\n > [Y/N] :{ENDC} ") or "n"
    if dec.upper() == "Y":
        for row_id in rows:
            cursor.execute("""DELETE from main WHERE rowid=? """, (row_id,))
        print(f"{BOLD} > '{len(rows)}' drop(s) {OKGREEN}successful{ENDC}")
    else:
        terminate()


def output(cursor):
    cursor.execute("SELECT * from main")
    data = cursor.fetchall()
    dec = input(
        f"{BOLD}\n > /path/to/write ?: default will be current working directory:{ENDC} "
    )
    with open(Path(dec) / "warp.out", "w", encoding="utf-8") as file:
        file.write("#" + ",".join(column_header(cursor)) + "\n")
        for conn in data:
            file.write(",".join(str(v) for v in conn) + "\n")


def terminate():
    print(f"\n{BOLD} > {WARNING}operation terminated{ENDC}")
    sys.exit(1)


def conn_close(_connect):
    _connect.commit()
    _connect.close()


def main():
    try:
        _connect, cursor = initialize_db()
        args = [key for key, value in vars(parse_args()).items() if value]
        if args:
            globals()[args[0]](cursor)
        else:
            print(f"{BOLD} > Use '-h' for options{ENDC}")
    except KeyboardInterrupt:
        print(f"{BOLD}\n > {FAIL}exit{ENDC}")
        sys.exit(1)
    conn_close(_connect)


if __name__ == "__main__":
    main()
