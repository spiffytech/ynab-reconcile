ynab-reconcile
==============

Reconciles a Mint data dump with a YNAB register

Usage
=====

Point the program at the directories where your YNAB and Mint exports live. It'll find the latest one of each.

`./yr.py --ynab-dir ~/ynab_exports --mint-dir ~/Downloads`

You can also point it at a specific file:

`./yr.py --ynab ~/ynab_export.csv --mint ~/Downloads/transactions.csv`
