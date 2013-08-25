package main

import (
    "encoding/csv"
    "fmt"
    "flag"
    "io"
    "os"
    "strconv"
    "strings"
    "time"
)

var ynabFilename = flag.String("ynab", "", "YNAB register export to use")
var mintFilename = flag.String("mint", "", "Mint.com register export to use")

func main() {
    flag.Parse()

    ynabFile, err := os.Open(*ynabFilename)
    if err != nil {
        panic(err)
    }
    ynab := new(YNAB)
    ynab.NewAccount(ynabFile)

    fmt.Println(ynab.Transactions()[0])

    mintFile, err := os.Open(*mintFilename)
    if err != nil {
        panic(err)
    }
    mint := new(Mint)
    mint.NewAccount(mintFile)

    fmt.Println(mint.Transactions()[0])
}


type Account interface {
    Name() string
    Transactions() *[]Transaction
    NewAccount(io.Reader)
}

type YNAB struct {
    _name string
    _transactions []Transaction
}
func (ynab *YNAB) Name() string {
    return ynab._name
}
func (ynab *YNAB) Transactions() []Transaction {
    return ynab._transactions
}

type Mint struct {
    _name string
    _transactions []Transaction
}
func (mint *Mint) Name() string {
    return mint._name
}
func (mint *Mint) Transactions() []Transaction {
    return mint._transactions
}

type Transaction interface {
    Date() time.Time
    Payee() string
    Category() string
    Amount() float64
}

type YNABTransaction struct {
    _Date time.Time
    _Payee string
    _Category string
    _Amount float64
    _Cleared bool
}

func (ynab *YNABTransaction) Date() time.Time {
    return ynab._Date
}
func (ynab *YNABTransaction) Payee() string {
    return ynab._Payee
}
func (ynab *YNABTransaction) Category() string {
    return ynab._Category
}
func (ynab *YNABTransaction) Amount() float64 {
    return ynab._Amount
}
func (ynab *YNABTransaction) Cleared() bool {
    return ynab._Cleared
}
func (ynab *YNAB) NewAccount(r io.Reader) {
    csvReader := csv.NewReader(r)
    csvReader.LazyQuotes = true
    headerLine := true
    for {
        line, err := csvReader.Read()
        if err != nil {
            if err == io.EOF {
                break
            }
            panic(err)
        }

        if headerLine == true {
            headerLine = false
            continue
        }

        t := new(YNABTransaction)
        //Mon Jan 2 15:04:05 -0700 MST 2006
        date, err := time.Parse("2006/01/02", line[3])
        if err != nil {
            panic(err)
        }
        t._Date = date
        t._Payee = line[4]
        t._Category = line[5]
        outflow, err := strconv.ParseFloat(strings.TrimPrefix(line[9], "$"), 64)
        if err != nil {
            panic(err)
        }
        if outflow == 0 {
            inflow, err := strconv.ParseFloat(strings.TrimPrefix(line[10], "$"), 64)
            if err != nil {
                panic(err)
            }
            t._Amount = inflow
        } else {
            t._Amount = outflow * -1
        }

        if line[11] == "U" {
            t._Cleared = false
        } else {
            t._Cleared = true
        }

        ynab._transactions = append(ynab._transactions, t)
    }
}

type MintTransaction struct {
    _Date time.Time
    _Description string
    _Category string
    _Amount float64
}

func (ynab *MintTransaction) Date() time.Time {
    return ynab._Date
}
func (ynab *MintTransaction) Payee() string {
    return ynab._Description
}
func (ynab *MintTransaction) Category() string {
    return ynab._Category
}
func (ynab *MintTransaction) Amount() float64 {
    return ynab._Amount
}

func (mint *Mint) NewAccount(r io.Reader) {
    csvReader := csv.NewReader(r)
    csvReader.LazyQuotes = true
    headerLine := true
    for {
        line, err := csvReader.Read()
        if err != nil {
            if err == io.EOF {
                break
            }
            panic(err)
        }

        if headerLine == true {
            headerLine = false
            continue
        }

        t := new(MintTransaction)
        //Mon Jan 2 15:04:05 -0700 MST 2006
        date, err := time.Parse("1/2/2006", line[0])
        if err != nil {
            panic(err)
        }
        t._Date = date
        t._Description = line[1]
        t._Category = line[5]
        amount, err := strconv.ParseFloat(line[3], 64)
        if err != nil {
            panic(err)
        }
        t._Amount = amount
        if line[4] == "debit" {
            t._Amount *= -1
        }

        mint._transactions = append(mint._transactions, t)
    }
}
