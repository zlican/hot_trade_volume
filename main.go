package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// CoinGlass URL
const coinglassURL = "https://www.coinglass.com/zh"

var (
	mu               sync.RWMutex
	symbols          []string
	topGainersSymbol []string
)

// fetchFilteredCoins 执行筛选并返回币种列表
func fetchFilteredCoins() ([]string, []string, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ProxyServer("http://127.0.0.1:10809"),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.WindowSize(1920, 1080),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 40*time.Second)
	defer cancel()

	var nodes []*cdp.Node
	var symbols []string

	err := chromedp.Run(ctx,
		network.Enable(),
		network.SetCookie("obe", "s_ce4eb2cfebfd4e24803c5078e4509ae9").WithDomain("www.coinglass.com").WithPath("/"),
		network.SetCookie("api_grade", "1").WithDomain("www.coinglass.com").WithPath("/"),

		chromedp.Navigate(coinglassURL),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),

		chromedp.Click(`button.home-custom-table-but:nth-of-type(1)`, chromedp.ByQuery),

		chromedp.WaitVisible(`//button[span[contains(text(),"成交额")]]`, chromedp.BySearch),
		chromedp.Click(`//button[span[contains(text(),"成交额")]]`, chromedp.BySearch),

		chromedp.Sleep(50*time.Millisecond),
		chromedp.Nodes(`//button[.//div[contains(text(),"24小时成交额")]]`, &nodes, chromedp.BySearch),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("初始页面操作失败: %v", err)
	}
	if len(nodes) < 2 {
		return nil, nil, fmt.Errorf("没有找到第二个“24小时成交额”按钮")
	}

	err = chromedp.Run(ctx,
		chromedp.WaitVisible(nodes[1].FullXPath()),
		chromedp.Click(nodes[1].FullXPath()),
		chromedp.Sleep(50*time.Millisecond),
		chromedp.Evaluate(`(() => {
			const inputs = document.evaluate(
				"//div[.//div[text()='24小时成交额']]//input[@placeholder='$0']",
				document,null,XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,null
			);
			if(inputs.snapshotLength >= 13){
				const input = inputs.snapshotItem(12);
				input.focus();
				const nativeSetter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype,'value').set;
				nativeSetter.call(input,'10000000');
				input.dispatchEvent(new Event('input',{bubbles:true}));
				input.dispatchEvent(new Event('change',{bubbles:true}));
				input.blur();
			}
		})()`, nil),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("设置24小时成交额失败: %v", err)
	}

	err = chromedp.Run(ctx,
		chromedp.Nodes(`//button[.//div[contains(text(),"成交额变化")]]`, &nodes, chromedp.BySearch),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("获取成交额变化按钮失败: %v", err)
	}
	if len(nodes) < 3 {
		return nil, nil, fmt.Errorf("没有找到3个成交额变化按钮")
	}

	err = chromedp.Run(ctx,
		chromedp.Sleep(50*time.Millisecond),
		chromedp.Evaluate(`(() => {
			const inputs = document.evaluate(
				"//div[.//div[text()='成交额变化(1小时)']]//input[@placeholder='-100%']",
				document,null,XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,null
			);
			const idxArr=[12,13,14];
			idxArr.forEach(i=>{
				if(inputs.snapshotLength>i){
					const input=inputs.snapshotItem(i);
					input.focus();
					const nativeSetter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype,'value').set;
					nativeSetter.call(input,'5');
					input.dispatchEvent(new Event('input',{bubbles:true}));
					input.dispatchEvent(new Event('change',{bubbles:true}));
					input.blur();
				}
			});
		})()`, nil),

		chromedp.Sleep(50*time.Millisecond),
		chromedp.Click(`//button[normalize-space(text())="应用筛选"]`, chromedp.NodeVisible),
		chromedp.Sleep(50*time.Millisecond),

		chromedp.Click(`//button[@role="combobox" and normalize-space(text())="20"]`, chromedp.NodeVisible),
		chromedp.Sleep(50*time.Millisecond),

		// 点击下拉中的 "100" 选项
		chromedp.Click(`//ul[@role="listbox"]//li[normalize-space(text())="100"]`, chromedp.NodeVisible),
		chromedp.Sleep(1000*time.Millisecond),

		chromedp.WaitVisible(`.ant-table-cell.ant-table-cell-fix-left-last`, chromedp.ByQuery),
		chromedp.Evaluate(`(() => {
			const nodes = document.querySelectorAll('.ant-table-cell.ant-table-cell-fix-left-last .symbol-name');
			return Array.from(nodes).map(n=>n.textContent.trim()).filter(v=>v && v!=="币种");
		})()`, &symbols),

		// 等待第一个结果块出现，确保 DOM 渲染完成
		chromedp.WaitVisible(`div.MuiBox-root.cg-style-1a7fd5v`, chromedp.ByQuery),
		// 抓取所有 symbol-name
		chromedp.Evaluate(`(() => {
		const nodes = document.querySelectorAll("div.MuiBox-root.cg-style-1a7fd5v .symbol-name");
		return Array.from(nodes).map(n => n.textContent.trim()).filter(v => v);
	})()`, &topGainersSymbol),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("应用筛选或获取币种失败: %v", err)
	}
	return symbols, topGainersSymbol, nil
}

// updateSymbols 定时刷新全局币种列表
func updateSymbols() {
	for {
		var newSymbols []string // 假设symbol是string类型，具体类型根据实际调整
		var newTopGainers []string
		var err error
		for attempt := 1; attempt <= 3; attempt++ {
			newSymbols, newTopGainers, err = fetchFilteredCoins()
			if err == nil {
				mu.Lock()
				symbols = newSymbols
				topGainersSymbol = newTopGainers
				mu.Unlock()
				log.Println("刷新币种成功", len(newSymbols), len(topGainersSymbol))
				break // 成功则跳出重试循环
			}
			log.Printf("第 %d 次刷新币种失败: %v", attempt, err)
			if attempt < 3 {
				time.Sleep(time.Second * time.Duration(attempt*2)) // 每次重试增加等待时间
			}
		}
		if err != nil {
			log.Printf("刷新币种失败，已重试3次: %v", err)
		}
		time.Sleep(5 * time.Minute) // 每5分钟刷新一次
	}
}

// hotTradeVolumeHandler 返回最新币种列表
func hotTradeVolumeHandler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(symbols)
}

// hotTradeVolumeHandler 返回最新币种列表
func topGainersHandler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(topGainersSymbol)
}

func main() {
	log.Println("启动 CoinGlass 爬取服务...")

	// 启动定时刷新协程
	go updateSymbols()

	// 创建 mux
	mux := http.NewServeMux()
	mux.HandleFunc("/api/hot_trade_volume", hotTradeVolumeHandler)
	mux.HandleFunc("/api/top_gainers", topGainersHandler)

	// 使用 CORS 中间件
	handler := corsMiddleware(mux)

	log.Println("HTTP服务启动，监听端口9000")
	log.Fatal(http.ListenAndServe(":9000", handler))
}
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
