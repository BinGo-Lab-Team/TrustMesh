package consensus

import "time"

// TimeRoundEngine 根据目标轮次与实际轮次对齐执行时机
func TimeRoundEngine(interval time.Duration, targetRound int64) chan int64 {
	ch := make(chan int64)

	go func() {
		intervalMs := interval.Milliseconds()

		for {
			realRound := time.Now().UnixMilli() / intervalMs

			if targetRound <= realRound {
				// 目标轮次已过期或正在当前轮 → 直接用当前轮
				ch <- realRound
				return
			}

			if targetRound == realRound+1 {
				// 目标刚好等于下一轮 → 提前启动
				ch <- targetRound
				return
			}

			// 目标超前超过1轮 → 等到下轮开始
			nextTime := time.UnixMilli((realRound + 1) * intervalMs)
			time.Sleep(time.Until(nextTime))
		}
	}()

	return ch
}

// TimeNextRoundComing 现在是否已经是目标轮次
func TimeNextRoundComing(interval time.Duration, targetRound int64) bool {
	round := time.Now().UnixMilli() / interval.Milliseconds()
	if round < targetRound {
		return false
	}

	return true
}

// TimeNextRound 到达目标轮次后 close chan
func TimeNextRound(interval time.Duration, targetRound int64) chan struct{} {
	out := make(chan struct{})

	go func() {
		defer close(out)

		round := time.Now().UnixMilli() / interval.Milliseconds()
		if round < targetRound {
			targetTime := time.UnixMilli(targetRound * interval.Milliseconds())
			timer := time.NewTimer(time.Until(targetTime))
			defer timer.Stop()

			select {
			case <-timer.C:
				return
			}
		} else {
			return
		}
	}()

	return out
}
