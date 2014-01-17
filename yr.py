#!/bin/env python

from __future__ import division
import argparse
import csv
from datetime import datetime, timedelta
import re
from pprint import pprint


parser = argparse.ArgumentParser(description="Compare a YNAB register with Mint ledger")
parser.add_argument("--ynab", help="Path to CSV containing YNAB register transactions")
parser.add_argument("--mint", help="Path to CSV containing Mint ledger")
parser.add_argument("--start-date", help="Path to CSV containing Mint ledger")

args = parser.parse_args()
if args.start_date:
    start_date = datetime.strptime(args.start_date, "%Y-%m-%d")
else:
    start_date = datetime.strptime("1970-01-01", "%Y-%m-%d")

def main():
    ynab = Ynab(args.ynab)
    mint = Mint(args.mint)

    pair_transactions(ynab, mint)
    ynab.transactions = [t for t in ynab.transactions if t.date.date() >= start_date.date()]
    mint.transactions = [t for t in mint.transactions if t.date.date() >= start_date.date()]

    separator = "------------------------------------"

    print separator
    ynab.print_unmatched_transactions()
    print separator
    mint.print_unmatched_transactions()
    print separator


def pair_transactions(a, b):
    for trans_a in a.transactions:
        for trans_b in b.transactions:
            if trans_a == trans_b and not trans_a.paired and not trans_b.paired:
                trans_a.paired = trans_b
                trans_b.paired = trans_a


class Transaction(object):
    def __init__(self, **kwargs):
        self.paired = False
        #self.date = kwargs["date"]
        #self.payee = kwargs["payee"]
        #self.category = kwargs["category"]
        #self.amount = kwargs["amount"]
        #self.cleared = kwargs["cleared"]

    def __str__(self):
        return "%s %s %s" % (self.date, self.amount, self.payee)


    def __eq__(self, other):
        if not isinstance(other, Transaction):
            return False

        return (self.date.date() <= other.date.date() <= self.date.date() + timedelta(days=5)) and (self.amount == other.amount)


    def __cmp__(self, other):
        return cmp(self.date.date(), other.date.date())


class Account(object):
    def print_unmatched_transactions(self):
        for transaction in (t for t in self.transactions if t.paired == False):
            print "%s - %s : %.2f (%s)" % (self.name, transaction.date, transaction.amount, transaction.payee)



class Ynab(Account):
    def __init__(self, filename):
        self.name = "YNAB" 
        self.transactions = []

        with open(filename) as register:
            dr = csv.DictReader(register)
            for row in dr:
                trans = self._process_row(row)
                while True:  # Merge split transactions into a single transaction
                    regex = r'\(Split ([0-9]+)/([0-9]+)\)'
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

                trans.amount = round(trans.amount, 2)
                self.transactions.append(trans)

        self.transactions.sort()
        #pprint([(t.date, t.amount, t.payee) for t in self.transactions])



    def _process_row(self, row):
        trans = Transaction()
        trans.date = datetime.strptime(row["Date"], "%Y/%m/%d")
        trans.payee = row["Payee"]
        trans.category = row["Category"]

        debit = float(row["Outflow"].strip("$"))
        if debit != 0:
            trans.amount = debit * -1
        else:
            credit = float(row["Inflow"].strip("$"))
            trans.amount = credit

        trans.cleared = (row["Cleared"] != "U")

        return trans



class Mint(Account):
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
        trans.date = datetime.strptime(row["Date"], "%m/%d/%Y")
        trans.payee = row["Description"]
        trans.raw_payee = row["Original Description"]
        trans.category = row["Category"]

        trans.amount = float(row["Amount"])
        if row["Transaction Type"] == "debit":
            trans.amount *= -1

        return trans


if __name__ == "__main__":
    main()
