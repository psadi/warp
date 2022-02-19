#!/usr/bin/env python3

"""
   Warp! - Your lazy ssh command line helper
"""
__version__ = '0.1.4'
__author__ = 'P S, Adithya (adithya3494@gmail.com)'
__license__ = 'MIT'

#standard imports
import os
import sys
import time
import sqlite3
import argparse
from pathlib import Path

#depends
from iterfzf import iterfzf

# globals
HEADER = '\033[95m'
OKBLUE = '\033[94m'
OKCYAN = '\033[96m'
OKGREEN = '\033[92m'
WARNING = '\033[93m'
FAIL = '\033[91m'
ENDC = '\033[0m'
BOLD = '\033[1m'

def parse_args():
    """Argument parsor returns string"""
    parser = argparse.ArgumentParser()
    p_group = parser.add_mutually_exclusive_group()
    p_group.add_argument('-a', '--add', help="add connection(s)", action="store_true")
    p_group.add_argument('-c', '--connect', help="initate a connection from stored values", action="store_true")
    p_group.add_argument('-d', '--delete', help="delete connection(s)", action="store_true")
    p_group.add_argument('-s', '--show', help="show all data", action="store_true")
    p_group.add_argument('-o', '--output', help="write out existing data to a file", action="store_true")
    # TO:DO, add filter, ping and alias functionality
    # parser.add_argument('--filter', help="interactive filter", action="store_true")
    # parser.add_argument('--ping', help="test your connection", action="store_true")
    # parser.add_argument('--alias', help="create an alias for a connection", action="store_true")
    return parser.parse_args()


def init(cursor):
    """Create a main table if it dosent exists"""
    cursor.execute(
        """CREATE TABLE if not exists main(environment text,hostname text,ip_address real
        ,username text,password text)""")
    cursor.execute(
        """CREATE TABLE if not exists alias(name text,ip_address real,username text,password text)""")


def initialize_db():
    """Check for db file in /home/$USER/.warp/warp.db -> create if dosent exist and init database"""
    config_dir = str(os.path.join(str(Path.home()), ".config/warp"))
    if not os.path.exists(config_dir):
        os.mkdir(config_dir)
    db_file = os.path.join(config_dir, 'warp.db')
    _connect = sqlite3.connect(db_file)
    cursor = _connect.cursor()
    init(cursor)
    return [_connect, cursor]


def column_header(cursor):
    """Get all column header from the main table"""
    return [description[0] for description in cursor.execute('select * from main').description]


def column_separator(header):
    """Create a string seprator for the length of each column header"""
    return [('-' * len(h)) for h in header]


def pretty_print(data, header, separator):
    """A fancy way to print all data in even row and columns based on string length"""
    widths = [len(cell) for cell in header]
    for row in data:
        for i, cell in enumerate(row):
            widths[i] = max(len(str(cell)), widths[i])
    formatted_row = ' '.join('{:%d}' % width for width in widths)
    print('\t' + formatted_row.format(*header))
    print('\t' + formatted_row.format(*separator))
    for row in data:
        print('\t' + formatted_row.format(*row))


def insert_func(cursor, data):
    """Inter a connection string into the database"""
    cursor.executemany("INSERT INTO main VALUES (?,?,?,?,?)", data)
    print(f"{BOLD} *> '{len(data)}' insert(s) {OKGREEN}successful{ENDC}")


def add(cursor):
    """
        Adds a connection, calls the insert function, Can enter manually or load from a file
        Refer: path-to-file-template
    """
    header = column_header(cursor)
    separator = column_separator(header)
    print(f'{BOLD} *>  Do you have a file to load ?{ENDC}')
    dec = input(f'{BOLD} *> [Y/N] :{ENDC} ') or 'n'
    if dec.upper() == 'N':
        print(f" *> Enter the details in '{BOLD}{','.join(header)}{ENDC}' format")
        a_inpt = [input(' *> INPUT: ').split(',')]
    elif dec.upper() == 'Y':
        print(f'{BOLD} *> /path/to/file ?{ENDC}')
        try:
            with open(input(), 'r', encoding='utf-8') as file:
                a_inpt = [line.strip('\n').split(',') for line in file if not line.startswith('#')]
        except FileNotFoundError:
            print(f'{BOLD} *> {FAIL}file not found{ENDC}')
            sys.exit(1)
    else:
        terminate()
    data = [tuple(d) for d in a_inpt]
    print(f'{BOLD} *> The below info will be added, proceed ?{ENDC}\n')
    pretty_print(data, header, separator)
    dec = input(f'{BOLD}\n *> [Y/N] :{ENDC} ')
    if dec.upper() == 'Y':
        insert_func(cursor, data)
    else: terminate()


def iter_data(data):
    formatted_data = []
    for d in data:
        formatted_data.append(' '.join(d))
    return formatted_data

def fzf_prompt(data):
    """Making use of pyfzf library to display the queried data"""

    try:
        return iterfzf(iter_data(data)).replace(' ',',').split(',')
        # return fzf.prompt(data, '--`multi --cycle')[0].translate(str.maketrans(
        #         {"(": "", ")": "", "'": "", " ": ""})).split(',')
    except Exception as e:
        print(e)
        terminate()

def connect(cursor):
    """Connect to a host based on the selected connection string"""
    try:
        cursor.execute("SELECT * from main")
        conn_data = fzf_prompt(cursor.fetchall())
        ssh_command = "ssh" + " " + conn_data[3] + "@" + conn_data[2]
        print(f"{BOLD} *> Executing {OKCYAN}'{ssh_command}'{ENDC}")
        os.system(ssh_command)
    except KeyboardInterrupt:
        terminate()


def show(cursor) -> str:
    """Show all available data in maintable, Pretty prints to console"""
    header = column_header(cursor)
    header.insert(0, 'ID')
    separator = column_separator(header)
    cursor.execute(f"SELECT rowid, * from main")
    data = cursor.fetchall()
    pretty_print(data, header, separator)


def delete(cursor):
    """
        Delete a connection string
        - can be a single value(1)
        - comma seprated value(1,2,3)
        - range of values(1_10)
    """
    header = column_header(cursor)
    header.insert(0, 'ID')
    separator = column_separator(header)
    print(f" *> Enter the rowid's to drop in {BOLD}id1,id2 or range:'id3_id6'{ENDC} format")
    d_inpt = input(' *> INPUT: ').split(',')
    rows = []
    try:
        for i in d_inpt:
            if '_' in i:
                if int(i.split('_')[1]) > int(i.split('_')[0]):
                    _ = [rows.append(str(_n))
                        for _n in range(int(i.split('_')[0]), int(i.split('_')[1]) + 1)]
                else:
                    _ = [rows.append(str(_n))
                    for _n in range(int(i.split('_')[0]), int(i.split('_')[1]) - 1, -1)]
            else: rows.append(i)
        data = []
        for row_id in rows:
            cursor.execute("""SELECT rowid, * from main WHERE rowid = ?""", (row_id,),)
            data.append(cursor.fetchall()[0])
    except IndexError:
        print(f"\n{BOLD} *> {WARNING}Invalid range, use '-s' to validate{ENDC}")
        sys.exit(1)
    print(f'{BOLD} *> Below data would be deleted... proceed ?{ENDC}\n')
    pretty_print(data, header, separator)
    dec = input(f'{BOLD}\n *> [Y/N] :{ENDC} ') or 'n'
    if dec.upper() == 'Y':
        for row_id in rows:
            cursor.execute(
                '''DELETE from main WHERE rowid=? ''', (row_id,))
        print(f"{BOLD} *> '{len(rows)}' drop(s) {OKGREEN}successful{ENDC}")
    else:
        terminate()


def output(cursor):
    """Output all data to a file, Sepecify a path, Default: current dir"""
    cursor.execute("SELECT * from main")
    data = cursor.fetchall()
    dec = input(f'{BOLD}\n *> /path/to/write ?: default will be current working directory:{ENDC} ')
    with open(os.path.join(f'{dec}','warp.out'), 'w', encoding="utf-8") as file:
        file.write('#' + ','.join(column_header(cursor)) + '\n')
        for conn in data:
            file.write(','.join(str(v) for v in conn) + '\n')

# def filter(cursor):
#     """
#         Interactive filter
#         //TO-DO - based on selection either show option to connect,delete or write to file
#     """
#     try:
#         print(column_header(cursor))
#         data = fzf_prompt(column_header(cursor))
#         cursor.execute("select DISTINCT " + data + " from main;")
#         selection = fzf_prompt(cursor.fetchall())
#         cursor.execute(f"SELECT * from main where {data}='{selection}'")
#         fzf_prompt(cursor.fetchall())
#     except KeyboardInterrupt:
#         terminate()


def terminate():
    """Something went wrong, exit with code 1"""
    print(f'\n{BOLD} *> {WARNING}operation terminated{ENDC}')
    sys.exit(1)


def conn_close(_connect):
    """ commit changes to databse and close connetion"""
    _connect.commit()
    _connect.close()


def main():
    """ JC main """
    try:
        sqlconn = initialize_db()
        (_connect, cursor) = (sqlconn[0], sqlconn[1])
        args = [key for key, value in vars(parse_args()).items() if value]
        if len(args) > 0:
            globals()[args[0]](cursor)
        else:
            print(f"{BOLD} *> Use \'-h\' for options{ENDC}")
    except KeyboardInterrupt:
        print(f'{BOLD}\n *> {FAIL}exit{ENDC}')
        sys.exit(1)
    conn_close(_connect)


if __name__ == "__main__":
    main()
