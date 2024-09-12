package main

/*
随机性测试：
1. 创建新的一轮游戏： new round
2. 下注：bet
3. crash游戏： crash
4. 统计：crash乘数分布
*/

type StatisticalData struct {
	RoundNum      uint64
	Seed          uint64
	CrashMultiple uint64
}

//func TestRandomness(t *testing.T) {
//	flag.Parse()
//	GetTonAPIIns()
//
//	f, err := os.OpenFile("log.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm)
//	if err != nil {
//		return
//	}
//	defer func() {
//		f.Close()
//	}()
//
//	// 组合一下即可，os.Stdout代表标准输出流
//	multiWriter := io.MultiWriter(os.Stdout, f)
//	log.SetOutput(multiWriter)
//	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
//
//	maxRoundNum := 10
//	count := 10
//	csvFile, err := os.Create("crash_multiple_1.csv")
//	defer csvFile.Close()
//	// 文件头
//	csvwriter := csv.NewWriter(csvFile)
//	csvwriter.Write([]string{"RoundNum", "Seed", "CrashMultiple(%)"})
//	if err != nil {
//		log.Fatalf("failed creating file: %s", err)
//	}
//	for roundNumIndex := 0; roundNumIndex <= maxRoundNum; roundNumIndex++ {
//		oneRowData := StatisticalData{}
//		// 获取当前游戏轮数
//		err, crashData := GetCrashGameData(*crashGameAddr, false)
//		if err != nil {
//			t.Error(err)
//			return
//		}
//		if 0 == roundNumIndex {
//			maxRoundNum = int(crashData.CurrentRoundNum) + count
//		}
//		roundNumIndex = int(crashData.CurrentRoundNum)
//		// 1. 创建新的一轮游戏： new round
//		if crashData.GameState == 1 {
//			log.Println("start new round ===================================", roundNumIndex)
//			err := NewRound(*crashGameAddr)
//			if err != nil {
//				t.Error(err)
//				return
//			}
//			log.Println("end new round ===================================", roundNumIndex)
//			crashData.CurrentRoundNum = crashData.CurrentRoundNum + 1
//		}
//		oneRowData.RoundNum = crashData.CurrentRoundNum
//
//		// 保存当前游戏轮数
//		roundNumIndex = int(crashData.CurrentRoundNum)
//		log.Println("roundNumIndex:", roundNumIndex)
//		// 2. 下注：bet
//		log.Println("start bet ===================================", roundNumIndex)
//		err = Bet(1, *crashGameAddr, *betAmount, *betMultiple)
//		if err != nil {
//			t.Error(err)
//			return
//		}
//		for {
//			// 获取当前游戏轮数
//			err, crashData = GetCrashGameData(*crashGameAddr, false)
//			if err != nil {
//				t.Error(err)
//				return
//			}
//			if crashData.PlayerNums > 0 {
//				break
//			}
//			time.Sleep(2 * time.Second)
//		}
//
//		log.Println("end bet ===================================", roundNumIndex)
//		// 3. crash游戏： crash
//		log.Println("start crash ===================================", roundNumIndex)
//		err = Crash(*crashGameAddr, uint64(roundNumIndex))
//		if err != nil {
//			t.Error(err)
//			return
//		}
//		log.Println("end crash ===================================", roundNumIndex)
//		for {
//			// 获取当前游戏轮数
//			err, crashData = GetCrashGameData(*crashGameAddr, false)
//			if err != nil {
//				t.Error(err)
//				return
//			}
//			if crashData.Seed > 0 {
//				break
//			}
//			time.Sleep(2 * time.Second)
//		}
//		// 获取当前游戏的seed和crash乘数
//		log.Printf("RoundNum:%d, Crash Seed:%d, CrashMultiple:%d\n", crashData.CurrentRoundNum, crashData.Seed, crashData.CrashMultiple)
//		oneRowData.Seed = crashData.Seed
//		oneRowData.CrashMultiple = crashData.CrashMultiple
//		var row []string
//		row = append(row, fmt.Sprintf("%d", oneRowData.RoundNum))
//		row = append(row, fmt.Sprintf("%d", oneRowData.Seed))
//		row = append(row, fmt.Sprintf("%d", oneRowData.CrashMultiple))
//		csvwriter.Write(row)
//	}
//	csvwriter.Flush()
//	err = csvwriter.Error()
//	if err != nil {
//		log.Fatalf("CSV writer error: %v", err)
//	}
//	log.Println("write to csv file success")
//}
