package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chromedp/cdproto/cdp"
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
	var nodes []*cdp.Node
	// 打开页面，设置 cookie
	err := chromedp.Run(ctx,
		network.Enable(),
		network.SetCookie("obe", "s_ce4eb2cfebfd4e24803c5078e4509ae9").
			WithDomain("www.coinglass.com").WithPath("/"),
		network.SetCookie("api_grade", "1").
			WithDomain("www.coinglass.com").WithPath("/"),

		chromedp.Navigate(coinglassURL),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),

		// 点击第一个按钮
		chromedp.Click(`button.home-custom-table-but:nth-of-type(1)`, chromedp.ByQuery),

		// 点击“成交额”按钮
		chromedp.WaitVisible(`//button[span[contains(text(),"成交额")]]`, chromedp.BySearch),
		chromedp.Click(`//button[span[contains(text(),"成交额")]]`, chromedp.BySearch),

		chromedp.Sleep(1*time.Second),

		// 获取所有“24小时成交额”按钮
		chromedp.Nodes(`//button[.//div[contains(text(),"24小时成交额")]]`, &nodes, chromedp.BySearch),
	)
	if err != nil {
		log.Fatal(err)
	}
	if len(nodes) < 2 {
		log.Fatal("没有找到第二个“24小时成交额”按钮")
	}

	// 修改“24小时成交额”输入框
	err = chromedp.Run(ctx,
		chromedp.WaitVisible(nodes[1].FullXPath()),
		chromedp.Click(nodes[1].FullXPath()),
		chromedp.Sleep(500*time.Millisecond),

		chromedp.Evaluate(`
	(() => {
		const inputs = document.evaluate(
			"//div[.//div[text()='24小时成交额']]//input[@placeholder='$0']",
			document,
			null,
			XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
			null
		);
		if (inputs.snapshotLength >= 13) {
			const input = inputs.snapshotItem(12);
			input.focus();
			const nativeSetter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype,'value').set;
			nativeSetter.call(input,'50000000');
			input.dispatchEvent(new Event('input',{bubbles:true}));
			input.dispatchEvent(new Event('change',{bubbles:true}));
			input.blur();
		}
	})()
	`, nil),
	)
	if err != nil {
		log.Fatal(err)
	}

	// 获取“成交额变化”按钮
	err = chromedp.Run(ctx,
		chromedp.Nodes(`//button[.//div[contains(text(),"成交额变化")]]`, &nodes, chromedp.BySearch),
	)
	if err != nil {
		log.Fatal(err)
	}
	if len(nodes) < 3 {
		log.Fatal("没有找到3个成交额变化按钮")
	}

	// 修改“成交额变化(1小时)”输入框，三个一起处理
	err = chromedp.Run(ctx,
		chromedp.Sleep(500*time.Millisecond),
		chromedp.Evaluate(`
	(() => {
		const inputs = document.evaluate(
			"//div[.//div[text()='成交额变化(1小时)']]//input[@placeholder='-100%']",
			document,
			null,
			XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
			null
		);
		const idxArr = [12,13,14];
		idxArr.forEach(i=>{
			if(inputs.snapshotLength>i){
				const input = inputs.snapshotItem(i);
				input.focus();
				const nativeSetter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype,'value').set;
				nativeSetter.call(input,'5');
				input.dispatchEvent(new Event('input',{bubbles:true}));
				input.dispatchEvent(new Event('change',{bubbles:true}));
				input.blur();
			}
		});
	})()
	`, nil),

		chromedp.Sleep(500*time.Millisecond),

		// 点击“应用筛选”
		chromedp.Click(`//button[normalize-space(text())="应用筛选"]`, chromedp.NodeVisible),

		// 等待表格加载
		chromedp.Sleep(2*time.Second),
		chromedp.WaitVisible(`.ant-table-cell.ant-table-cell-fix-left-last`, chromedp.ByQuery),

		// 提取币种
		chromedp.Evaluate(`
	(() => {
		const nodes = document.querySelectorAll('.ant-table-cell.ant-table-cell-fix-left-last .symbol-name');
		return Array.from(nodes).map(n=>n.textContent.trim()).filter(v=>v && v!=="币种");
	})()
	`, &symbols),
	)
	if err != nil {
		log.Fatal(err)
	}
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
