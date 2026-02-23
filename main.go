package main

import (
	"fmt"
	"io/fs"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// Bag Of Words
type Bow map[string]int
type MailType string

const (
	Ham  MailType = "ham"
	Spam MailType = "spam"
)

func tokenizeFile(data string) []string {
	words := strings.FieldsFunc(string(data), func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r) || !unicode.IsLetter(r)
	})

	return words
}

func addWordsToBow(bow Bow, words []string) {
	for _, word := range words {
		bow[word] += 1
	}
}

func addFileToBow(path string, bow Bow) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	words := tokenizeFile(string(data))
	addWordsToBow(bow, words)

	return nil
}

func processDirToBow(path string, bow Bow) error {
	fmt.Printf("Processing %s\n", path)
	return filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		err = addFileToBow(path, bow)
		if err != nil {
			return err
		}

		return nil
	})
}

func processDirsToBow(mailType MailType, bow Bow) error {
	for i := 1; i <= 5; i++ {
		err := processDirToBow(fmt.Sprintf("./enron%d/%s/", i, mailType), bow)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Processing %s complete!\n", mailType)

	return nil
}

func totalWordCount(bow Bow) int {
	count := 0
	for token := range bow {
		count += bow[token]
	}

	return count
}

type mailCategorizer struct {
	hamBow         Bow
	spamBow        Bow
	totalHamCount  int
	totalSpamCount int
	totalCount     int
}

func newMailCategorizer(hamBow, spamBow Bow, totalHamCount, totalSpamCount int) mailCategorizer {
	return mailCategorizer{
		hamBow:         hamBow,
		spamBow:        spamBow,
		totalHamCount:  totalHamCount,
		totalSpamCount: totalSpamCount,
		totalCount:     totalHamCount + totalSpamCount,
	}
}

func (mc *mailCategorizer) categorizeMail(path string) (float64, float64, error) {
	emailBow := Bow{}

	err := addFileToBow(path, emailBow)
	if err != nil {
		return 0, 0, err
	}

	pDHam := 0.0
	pDSpam := 0.0
	pD := 0.0
	pHam := math.Log(float64(mc.totalHamCount) / float64(mc.totalCount))
	pSpam := math.Log(float64(mc.totalSpamCount) / float64(mc.totalCount))

	for word := range emailBow {
		if mc.hamBow[word] == 0 || mc.spamBow[word] == 0 {
			continue
		}
		pDHam += math.Log(float64(mc.hamBow[word]) / float64(mc.totalHamCount))
		pDSpam += math.Log(float64(mc.spamBow[word]) / float64(mc.totalSpamCount))
		pD += math.Log(float64(mc.hamBow[word]+mc.spamBow[word]) / float64(mc.totalCount))
	}

	ham := pDHam + pHam - pD
	spam := pDSpam + pSpam - pD

	return ham, spam, nil
}

func (mc mailCategorizer) categorizeMails(mailType MailType) (int, int, error) {
	hamCount := 0
	spamCount := 0
	err := filepath.WalkDir(fmt.Sprintf("./enron6/%v/", mailType), func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		ham, spam, err := mc.categorizeMail(path)
		if err != nil {
			return err
		}

		if ham > spam {
			hamCount += 1
		} else {
			spamCount += 1
		}
		return nil
	})
	if err != nil {
		return 0, 0, err
	}

	return hamCount, spamCount, nil
}

func main() {
	hamBow := Bow{}
	spamBow := Bow{}

	err := processDirsToBow(Ham, hamBow)
	if err != nil {
		log.Fatal(err)
	}
	err = processDirsToBow(Spam, spamBow)
	if err != nil {
		log.Fatal(err)
	}

	totalHamCount := totalWordCount(hamBow)
	totalSpamCount := totalWordCount(spamBow)

	mc := newMailCategorizer(hamBow, spamBow, totalHamCount, totalSpamCount)

	hams, spams, err := mc.categorizeMails(Ham)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Categorized Ham:")
	fmt.Printf("  Hams: %v\n", hams)
	fmt.Printf("  Spams: %v\n", spams)

	hams, spams, err = mc.categorizeMails(Spam)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Categorized Spam:")
	fmt.Printf("  Hams: %v\n", hams)
	fmt.Printf("  Spams: %v\n", spams)
}
