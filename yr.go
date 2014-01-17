package main

import (
    "encoding/csv"
    "errors"
    "fmt"
    "flag"
    "io"
    "os"
    "regexp"
    "strconv"
    "strings"
    "time"
)

var ynabFilename = flag.String("ynab", "", "YNAB register export to use")
var mintFilename = flag.String("mint", "", "Mint.com register export to use")
var minDateStr = flag.String("startDate", "", "")
var minDate time.Time

func main() {
    flag.Parse()
    var err error
    if *minDateStr == "" {
        minDate = time.Now()
    } else {
        minDate, err = time.Parse("2006-01-02", *minDateStr)
        if err != nil {
            panic("Invalid start date")
        }
    }

    ynabFile, err := os.Open(*ynabFilename)
    if err != nil {
        panic(err)
    }
    ynab := new(YNAB)
    ynab.NewAccount(ynabFile)

    mintFile, err := os.Open(*mintFilename)
    if err != nil {
        panic(err)
    }
    mint := new(Mint)
    mint.NewAccount(mintFile)

    unpairedTransactions(ynab, mint)
}


func unpairedTransactions(register, bank Account) {
    for _, transaction := range *(register.Transactions()) {
        matchTrans(&transaction, bank.Transactions())
    }
    for _, account := range []Account{register, bank} {
        for _, transaction := range *(account.Transactions()) {
            if transaction.Paired() == false && transaction.Date().After(minDate) {
                fmt.Println(account.Name(), "-", transaction)
            }
        }
        fmt.Println("------------------------------------")
    }
}

func matchTrans(transaction *Transaction, comp *[]Transaction) (*Transaction, error) {
    /*
    if strings.Contains((*transaction).Payee(), "Subway") {
        fmt.Println("here", *transaction)
    }
    */
    for i, t := range *comp {
        if t.Paired() == true {
            continue
        }

        daysFromNow := (*transaction).Date().Add(time.Duration(5*24)*time.Hour)
        /*
        // Debug statements
        if t.Amount() == -28.01 {
            fmt.Println(t)
            fmt.Println("trans:", *transaction)
            fmt.Println("comp: ", t)
            fmt.Println(t.Amount() - (*transaction).Amount())
            fmt.Println(t.Amount() == (*transaction).Amount())
            if(t.Amount() == (*transaction).Amount()) {
                fmt.Println(t.Date().Before(daysFromNow) || t.Date() == daysFromNow)
            }
            //fmt.Println("comp:", daysFromNow)
        }
        */
        amount_difference := t.Amount() - (*transaction).Amount()
        if (
            (t.Date().Before(daysFromNow) || t.Date() == daysFromNow) &&
            (t.Date().After((*transaction).Date()) || t.Date() == (*transaction).Date())) && 
            (amount_difference > -.001 && amount_difference < .001) {
            (*comp)[i].Pair()
            (*transaction).Pair()
            if (*transaction).Amount() == -12.90 {
                //fmt.Println("Paired:", (*transaction).Paired())
            }
            return &t, nil
        }
    }

    return nil, errors.New("No match found")
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
func (ynab *YNAB) Transactions() *[]Transaction {
    return &ynab._transactions
}

type Mint struct {
    _name string
    _transactions []Transaction
}
func (mint *Mint) Name() string {
    return mint._name
}
func (mint *Mint) Transactions() *[]Transaction {
    return &mint._transactions
}

type Transaction interface {
    Date() time.Time
    Payee() string
    Category() string
    Amount() float64
    Paired() bool
    Pair()
}

type YNABTransaction struct {
    _Date time.Time
    _Payee string
    _Category string
    _Amount float64
    _Cleared bool
    _Paired bool
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
func (ynab *YNABTransaction) Paired() bool {
    return ynab._Paired
}
func (ynab *YNABTransaction) Pair() {
    ynab._Paired = true
    //fmt.Println("Pairing:", ynab)
}
func (ynab *YNAB) NewAccount(r io.Reader) {
    ynab._name = "YNAB"

    csvReader := csv.NewReader(r)
    csvReader.LazyQuotes = true

    headerLine := true
    usingPlaceholder := false
    placeholder := new(YNABTransaction)
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

        /*
        if strings.Contains(t._Payee, "Subway") == true {
            fmt.Println("found it:", t)
        }
        */

        if strings.Contains(line[8], "Split") == true {
            if usingPlaceholder == false{
                t._Category = "Split"
                placeholder = t
                usingPlaceholder = true
            } else {
                placeholder._Amount += t._Amount

                re := regexp.MustCompile("Split ([0-9]+)/([0-9]+)")
                matches := re.FindStringSubmatch(line[8])
                if matches[1] == matches[2] {
                    ynab._transactions = append(ynab._transactions, placeholder)
                    placeholder = new(YNABTransaction)
                    usingPlaceholder = false
                }
            }
        } else {
            ynab._transactions = append(ynab._transactions, t)
        }
    }
}

func (ynab *YNABTransaction) String() string {
    return fmt.Sprintf("%s: %.2f (%s)", ynab._Date.Format("2006-01-02"), ynab._Amount, ynab._Payee)
}

type MintTransaction struct {
    _Date time.Time
    _Description string
    _Category string
    _Amount float64
    _Paired bool
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
func (mint *MintTransaction) Paired() bool {
    return mint._Paired
}
func (mint *MintTransaction) Pair() {
    mint._Paired = true
}
func (mint *Mint) NewAccount(r io.Reader) {
    mint._name = "Mint"

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

func (mint *MintTransaction) String() string {
    return fmt.Sprintf("%s: %.2f (%s)", mint._Date.Format("2006-01-02"), mint._Amount, mint._Description)
}
