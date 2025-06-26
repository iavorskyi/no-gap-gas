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
	if email == "" || password == "" {
		log.Fatal("GASOLINA_EMAIL and GASOLINA_PASSWORD environment variables must be set")
	}

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var pageTitle string
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://gasolina-online.com/login"),
		chromedp.WaitVisible(`input[name="email"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="email"]`, email, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="password"]`, password, chromedp.ByQuery),
		chromedp.Click(`button[type="submit"]`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second), // wait for login to process
		chromedp.Navigate("https://gasolina-online.com/indicator"),
		chromedp.Title(&pageTitle),
	)
	if err != nil {
		log.Fatal("Failed to navigate to indicator page:", err)
	}

	var tableHTML string
	err = chromedp.Run(ctx,
		chromedp.OuterHTML(`.table-responsive`, &tableHTML, chromedp.ByQuery, chromedp.NodeVisible),
	)
	if err != nil {
		log.Fatal("Failed to get table HTML:", err)
	}

	// currentMonth := time.Now().Month()
	// isCurrentMonthMetricExists := false

	type Metric struct {
		Date         time.Time
		Value        int
		DeviceNumber string
		Description  string
	}

	// Parse tableHTML into []Metric
	var metrics []Metric

	type rowData struct {
		DeviceNumber string
		DateStr      string
		ValueStr     string
		Description  string
	}

	// Use regexp to extract <tr>...</tr> blocks from <tbody>
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

				// Parse date
				date, err := time.Parse("02.01.2006", dateStr)
				if err != nil {
					continue
				}
				// Parse value
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

	// Check if the last metric's date is between 25th of last month and 5th of current month
	actualMetricValue := ""
	if len(metrics) > 0 {
		now := time.Now()
		// Get the 25th of last month (date only, zero time)
		firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		lastMonth := firstOfThisMonth.AddDate(0, -1, 0)
		startDate := time.Date(lastMonth.Year(), lastMonth.Month(), 25, 0, 0, 0, 0, now.Location())
		// Get the 5th of this month (date only, zero time)
		endDate := time.Date(now.Year(), now.Month(), 5, 0, 0, 0, 0, now.Location())

		for _, metric := range metrics {
			metricDate := time.Date(metric.Date.Year(), metric.Date.Month(), metric.Date.Day(), 0, 0, 0, 0, metric.Date.Location())
			// Check if metricDate is between startDate and endDate (inclusive)
			if (metricDate.Equal(startDate) || metricDate.After(startDate)) && (metricDate.Equal(endDate) || metricDate.Before(endDate)) {
				fmt.Println("A metric date is between 25th of last month and 5th of current month")
				actualMetricValue = strconv.Itoa(metric.Value)
				break
			}
		}
		if actualMetricValue == "" {
			fmt.Println("No metric date is between 25th of last month and 5th of current month")
		}
	}

	fmt.Println("Metrics:", metrics)
	fmt.Println("Is actual metric exists:", actualMetricValue)

	if actualMetricValue != "" {
		ctx, cancel := chromedp.NewContext(context.Background())
		defer cancel()

		// Navigate to the page
		err := chromedp.Run(ctx,
			chromedp.Navigate("https://gasolina-online.com/"),
			// Wait for the button to be enabled (not disabled)
			chromedp.WaitNotPresent(`button.btn.material-btn.disabled`, chromedp.ByQuery),
		)
		if err != nil {
			fmt.Println("Error navigating or waiting for button:", err)
			return
		}

		// Click the enabled button
		err = chromedp.Run(ctx,
			chromedp.Click(`button.btn.material-btn`, chromedp.ByQuery),
		)
		if err != nil {
			fmt.Println("Error clicking the button:", err)
			return
		}

		// Calculate the new value to insert
		actualValueInt, err := strconv.Atoi(actualMetricValue)
		if err != nil {
			fmt.Println("Error converting actualMetricValue to int:", err)
			return
		}
		newValue := strconv.Itoa(actualValueInt + 20)

		// Insert the new value into the form
		// (Assuming the input field has a unique selector, e.g. input[name="value"] or similar)
		// You may need to adjust the selector below to match the actual form field
		err = chromedp.Run(ctx,
			chromedp.SetValue(`input[type="text"]`, newValue, chromedp.ByQuery),
		)
		if err != nil {
			fmt.Println("Error setting value in the form:", err)
			return
		}

		// Click the button
		err = chromedp.Run(ctx,
			chromedp.Click(`button.btn.material-btn`, chromedp.ByQuery),
		)
		if err != nil {
			fmt.Println("Error clicking the button:", err)
			return
		}
	}

}
