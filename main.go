package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	email := os.Getenv("EMAIL")
	password := os.Getenv("PASSWORD")
	// email := "7038329@ukr.net"
	// password := "Iavorskyi0938536997"
	if email == "" || password == "" {
		log.Fatal("GASOLINA_EMAIL and GASOLINA_PASSWORD environment variables must be set")
	}

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Login and navigate to indicator page
	var tableHTML string
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://gasolina-online.com/login"),
		chromedp.WaitVisible(`input[name="email"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="email"]`, email, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="password"]`, password, chromedp.ByQuery),
		chromedp.Click(`button[type="submit"]`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
		chromedp.Navigate("https://gasolina-online.com/indicator"),
		chromedp.OuterHTML(`.table-responsive`, &tableHTML, chromedp.ByQuery, chromedp.NodeVisible),
	)
	if err != nil {
		log.Fatal("Failed to login or get table HTML:", err)
	}

	type Metric struct {
		Date         time.Time
		Value        int
		DeviceNumber string
		Description  string
	}

	var metrics []Metric
	reTbody := regexp.MustCompile(`(?s)<tbody>(.*?)</tbody>`)
	reTr := regexp.MustCompile(`(?s)<tr>(.*?)</tr>`)
	reTd := regexp.MustCompile(`(?s)<td>(.*?)</td>`)

	tbodyMatch := reTbody.FindStringSubmatch(tableHTML)
	if len(tbodyMatch) > 1 {
		tbody := tbodyMatch[1]
		trs := reTr.FindAllStringSubmatch(tbody, -1)
		for _, tr := range trs {
			tds := reTd.FindAllStringSubmatch(tr[1], -1)
			if len(tds) >= 4 {
				deviceNumber := strings.TrimSpace(tds[0][1])
				dateStr := strings.TrimSpace(tds[1][1])
				valueStr := strings.TrimSpace(tds[2][1])
				description := strings.TrimSpace(tds[3][1])
				date, err := time.Parse("02.01.2006", dateStr)
				if err != nil {
					continue
				}
				value, err := strconv.Atoi(valueStr)
				if err != nil {
					continue
				}
				metrics = append(metrics, Metric{
					Date:         date,
					Value:        value,
					DeviceNumber: deviceNumber,
					Description:  description,
				})
			}
		}
	}

	// Check if a metric exists between 25th of last month and 5th of current month
	actualMetricExists := false
	if len(metrics) > 0 {
		now := time.Now()
		firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		lastMonth := firstOfThisMonth.AddDate(0, -1, 0)
		startDate := time.Date(lastMonth.Year(), lastMonth.Month(), 25, 0, 0, 0, 0, now.Location())
		endDate := time.Date(now.Year(), now.Month(), 5, 0, 0, 0, 0, now.Location())
		for _, metric := range metrics {
			metricDate := time.Date(metric.Date.Year(), metric.Date.Month(), metric.Date.Day(), 0, 0, 0, 0, metric.Date.Location())
			if (metricDate.Equal(startDate) || metricDate.After(startDate)) && (metricDate.Equal(endDate) || metricDate.Before(endDate)) {
				actualMetricExists = true
				break
			}
		}
	}

	fmt.Println("metrics", metrics)

	if !actualMetricExists {
		var previousValueStr string
		err = chromedp.Run(ctx,
			chromedp.Navigate("https://gasolina-online.com/"),
			chromedp.WaitVisible(`button.btn.material-btn.modal-indicator[data-toggle="modal"][data-target="#counterModal"]`, chromedp.ByQuery),
			chromedp.Click(`button.btn.material-btn.modal-indicator[data-toggle="modal"][data-target="#counterModal"]`, chromedp.ByQuery),
			chromedp.WaitVisible(`#counterModal #previous_value`, chromedp.ByQuery),
			chromedp.Value(`#counterModal #previous_value`, &previousValueStr, chromedp.ByQuery),
		)
		if err != nil {
			log.Fatal("Failed to get previous value:", err)
		}

		previousValue, err := strconv.Atoi(previousValueStr)
		if err != nil {
			log.Fatal("Error converting previous_value to int:", err)
		}
		newValueStr := strconv.Itoa(previousValue + 7)

		err = chromedp.Run(ctx,
			chromedp.SetValue(`#counterModal #value`, newValueStr, chromedp.ByQuery),
			chromedp.Click(`#counterModal button.btn.material-btn[type="submit"]`, chromedp.ByQuery),
		)
		if err != nil {
			log.Fatal("Error setting new value or clicking submit:", err)
		}
	}

	fmt.Println("SUCCESS")
}
