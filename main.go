package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// CoinGlass URL
const coinglassURL = "https://www.coinglass.com/zh"

// fetchFilteredCoins 执行筛选并返回币种列表
func fetchFilteredCoins() ([]string, error) {
	// 配置浏览器
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ProxyServer("http://127.0.0.1:10809"),
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.WindowSize(1920, 1080),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	// 设置超时
	ctx, cancel = context.WithTimeout(ctx, 40*time.Second)
	defer cancel()

	var symbols []string
	err := chromedp.Run(ctx,
		network.Enable(),
		network.SetCookie("obe", "s_ce4eb2cfebfd4e24803c5078e4509ae9").
			WithDomain("www.coinglass.com").WithPath("/"),
		network.SetCookie("api_grade", "1").
			WithDomain("www.coinglass.com").WithPath("/"),

		// 打开页面
		chromedp.Navigate(coinglassURL),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),

		// 点击第一个按钮
		chromedp.Click(`button.home-custom-table-but:nth-of-type(1)`, chromedp.ByQuery),

		// 点击“成交额”按钮
		chromedp.WaitVisible(`//button[span[contains(text(),"成交额")]]`, chromedp.BySearch),
		chromedp.Click(`//button[span[contains(text(),"成交额")]]`, chromedp.BySearch),
		//成功

		// 设置1小时成交额变化 >= 5
		chromedp.Click(`/html/body/div[3]/div[3]/div[2]/div[5]/div/div[2]/div/div[2]/div/div[1]/button/div`),
		chromedp.SetValue(`/html/body/div[3]/div[3]/div[2]/div[5]/div/div[2]/div/div[2]/div/div[2]/div/div[1]/div[1]/input`, "5"),

		// 设置4小时成交额变化 >= 5
		chromedp.Click(`/html/body/div[3]/div[3]/div[2]/div[5]/div/div[2]/div/div[3]/div/div[1]/button/div`),
		chromedp.SetValue(`/html/body/div[3]/div[3]/div[2]/div[5]/div/div[2]/div/div[3]/div/div[2]/div/div[1]/div[1]/input`, "5"),

		// 设置24小时成交额变化 >= 5
		chromedp.Click(`/html/body/div[3]/div[3]/div[2]/div[5]/div/div[2]/div/div[4]/div/div[1]/button/div`),
		chromedp.SetValue(`/html/body/div[3]/div[3]/div[2]/div[5]/div/div[2]/div/div[4]/div/div[2]/div/div[1]/div[1]/input`, "5"),

		// 点击“应用筛选”
		chromedp.Click(`/html/body/div[3]/div[3]/div[3]/div/div[2]/div/div[3]/button`),

		// 等待表格加载
		chromedp.Sleep(3*time.Second),
		chromedp.WaitVisible(`.ant-table-cell.ant-table-cell-fix-left-last`, chromedp.ByQuery),

		// 提取所有币种
		chromedp.Evaluate(`
			(() => {
				const nodes = document.querySelectorAll('.ant-table-cell.ant-table-cell-fix-left-last .symbol-name');
				return Array.from(nodes).map(n => n.textContent.trim()).filter(v => v && v !== "币种");
			})()
		`, &symbols),
	)

	if err != nil {
		return nil, fmt.Errorf("执行失败: %v", err)
	}
	return symbols, nil
}

func main() {
	log.SetOutput(os.Stdout)
	log.Println("启动第二个爬取脚本...")

	coins, err := fetchFilteredCoins()
	if err != nil {
		log.Fatalf("抓取失败: %v", err)
	}
	fmt.Println("符合条件的币种列表：", coins)
}
