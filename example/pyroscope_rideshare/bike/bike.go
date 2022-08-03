package bike

import "rideshare/utility"

func OrderBike(searchRadius int64) {
	utility.FindNearestVehicle(searchRadius, "bike")
	for i := 0; i < 3; i++ {
		go utility.AllocMem()
	}
}
