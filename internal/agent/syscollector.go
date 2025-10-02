package agent

import (
	"strconv"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// собирает дополнительные системные метрики
func (mc *MetricsCollector) CollectSys() {
	// Сначала собираем данные вне блокировок
	var total, free float64
	haveMem := false
	if vm, err := mem.VirtualMemory(); err == nil {
		total = float64(vm.Total)
		free = float64(vm.Free)
		haveMem = true
	}

	var percents []float64
	if p, err := cpu.Percent(0, true); err == nil {
		percents = p
	}

	// обновляем коллекцию метрик (внутренние методы берут блокировку)
	if haveMem {
		mc.updateGauge("TotalMemory", total)
		mc.updateGauge("FreeMemory", free)
	}
	for i, v := range percents {
		name := "CPUutilization" + strconv.Itoa(i+1)
		mc.updateGauge(name, v)
	}
}
