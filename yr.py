#!/usr/bin/env python

from __future__ import division
import argparse
import csv
from datetime import datetime, timedelta
import glob
import os
import re
from pprint import pprint
import dateutil.parser

parser = argparse.ArgumentParser(description="Compare a YNAB register with Mint ledger")

parser.add_argument("--ynab", help="Path to CSV containing YNAB register transactions")
parser.add_argument("--ynab-dir", help="Path to folder containing YNAB register in CSV format. Looks at latest register file.")

parser.add_argument("--mint", help="Path to CSV containing Mint ledger")
parser.add_argument("--mint-dir", help="Path to folder containing Mint CSV exports. Looks at latest export.")

parser.add_argument("--start-date", help="Path to CSV containing Mint ledger")

parser.add_argument("--account", help="Text name of YNAB account to filter for")

args = parser.parse_args()

def main():
    if args.start_date:
        start_date = dateutil.parser.parse(args.start_date)
    else:
        start_date = dateutil.parser.parse("1970-01-01")

    ynab_file, mint_file = pick_files(args)
    ynab = Ynab(ynab_file, args.account or None)
    mint = Mint(mint_file)

    first_bank_transaction = mint.transactions[0]

    pair_transactions(ynab, mint)

    ynab.transactions = [t for t in ynab.transactions if t.date.date() >= start_date.date() and t.date.date() >= first_bank_transaction.date.date()]
    mint.transactions = [t for t in mint.transactions if t.date.date() >= start_date.date()]

    separator = "------------------------------------"

    print separator
    ynab.print_unmatched_transactions()
    print separator
    mint.print_unmatched_transactions()
    print separator


def pick_files(args):
    if args.ynab_dir:
        ynab_file = Ynab.find_latest_file(args.ynab_dir)
    else:
        ynab_file = args.ynab
    assert ynab_file

    if args.mint:
        mint_file = args.mint
    else:
        mint_file = Mint.find_latest_file(args.mint_dir)
    assert mint_file


    return (ynab_file, mint_file)


def pair_transactions(a, b):
    # This is an O(2N) algorithm using dictionary keys for O(1) check for existance of a matching transaction.
    # Previously used a simple, but O(n^2), algorithm.
    # Treats (date, amount) as the key to look up transactions by
    # Includes support for multiple transactions hashing to the same key (e.g., you buy two TV shows on Google Play for $2.14 on the same day)
    seen_transactions = {}

    for trans_a in a.transactions:
        for key in trans_a.hash_keys():
            if key not in seen_transactions:
                seen_transactions[key] = []
            seen_transactions[key].append(trans_a)

    for trans_b in b.transactions:
        key = (trans_b.date.date(), trans_b.amount)
        if key not in seen_transactions:
            continue
        for transaction in seen_transactions[key]:
            if not transaction.is_paired():
                trans_b.pair(transaction)
                break


class Transaction(object):
    def __init__(self, **kwargs):
        self.paired = False
        self.timedelta = 5
        self.cleared = None  # Lets us skip printing cleared transactions as
                             # unmatched when we get reissued a new card that
                             # doesn't have all of the old transactions on it
                             # that our register includes, since we don't create
                             # new registers for new issuances of the same card

    def __str__(self):
        return "%s %s %s" % (self.date.strftime("%Y-%m-%d"), self.amount, self.payee)


    def __eq__(self, other):
        if not isinstance(other, Transaction):
            return False

        return (self.date.date() <= other.date.date() <= self.date.date() + timedelta(days=self.timedelta)) and (self.amount == other.amount)


    def hash_keys(self):
        """Dict keys this transaction would match, for pairing purposes"""
        keys = []
        for d in (self.date.date() + timedelta(delta) for delta in range(self.timedelta+1)):
            keys.append((d, self.amount))

        return keys


    def __cmp__(self, other):
        return cmp(self.date.date(), other.date.date())


    def pair(self, other):
        self.paired = other
        other.paired = self


    def is_paired(self):
        return self.paired is not False


class Account(object):
    def print_unmatched_transactions(self):
        #for transaction in (t for t in self.transactions if t.paired is False and t.cleared is not True):  # Must check for cleared to be either False and None.
        for transaction in (t for t in self.transactions if t.paired is False):  # Must check for cleared to be either False and None.
            print "%s - %s : %+.2f (%s)" % (self.name, transaction.date.strftime("%Y-%m-%d"), transaction.amount, transaction.payee)



class Ynab(Account):
    @staticmethod
    def find_latest_file(dir_):
        files = glob.glob(os.path.expanduser(os.path.join(dir_, "My Budget as of *-*Register.csv")))
        return sorted(files, key=os.path.getmtime)[-1]


    def __init__(self, filename, account=None):
        self.name = "YNAB"
        self.transactions = []

        with open(filename) as register:
            dr = csv.DictReader(register)
            for row in dr:
                trans = self._process_row(row)

                # Filter for only the specified account. Thanks, New YNAB.
                if account and trans.account != account:
                    continue

                while True:  # Merge split transactions into a single transaction
                    regex = r'Split \(([0-9]+)/([0-9]+)\)'
                    match = re.match(regex, row["Memo"])
                    if not match:
                        break

                    for split_row in dr:
                        match = re.match(regex, split_row["Memo"])
                        t = self._process_row(split_row)
                        trans.amount += t.amount

                        current_split = match.group(1)
                        max_splits = match.group(2)
                        if current_split == max_splits:
                            break
                    break

                trans.amount = round(trans.amount, 2)  # This fixes errors from adding numbers that can't be represented in binary and expecting them to equal one that can that came from Mint.
                self.transactions.append(trans)

        self.transactions.sort()
        #pprint([(t.date, t.amount, t.payee) for t in self.transactions])


    def _process_row(self, row):
        trans = Transaction()
        trans.date = dateutil.parser.parse(row["Date"])
        trans.payee = row["Payee"]
        trans.category = row["Category"]
        trans.cleared = row["Cleared"] == "C"  # C/U Cleared/Uncleared
        trans.account = row['\xef\xbb\xbf"Account"']

        debit = float(row["Outflow"].strip("$"))
        if debit != 0:
            trans.amount = debit * -1
        else:
            credit = float(row["Inflow"].strip("$"))
            trans.amount = credit

        trans.cleared = (row["Cleared"] != "U")

        return trans



class Mint(Account):
    @staticmethod
    def find_latest_file(dir_):
        files = glob.glob(os.path.expanduser(os.path.join(dir_, "transactions*.csv")))
        return sorted(files, key=os.path.getmtime)[-1]


    def __init__(self, filename):
        self.name = "Mint"
        self.transactions = []

        with open(filename) as ledger:
            dr = csv.DictReader(ledger)
            for row in dr:
                trans = self._process_row(row)

                self.transactions.append(trans)

        self.transactions.sort()
        #pprint([(t.date, t.amount, t.payee) for t in self.transactions])


    def _process_row(self, row):
        trans = Transaction()
        trans.date = dateutil.parser.parse(row["Date"])
        trans.payee = row["Description"]
        trans.raw_payee = row["Original Description"]
        trans.category = row["Category"]

        trans.amount = float(row["Amount"])
        if row["Transaction Type"] == "debit":
            trans.amount *= -1

        return trans


if __name__ == "__main__":
    main()
