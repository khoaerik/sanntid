package queue

import "../elevio"
//import "fmt"

type Order struct {
	Pushed elevio.ButtonEvent
	//ip network.IPc
	// IsDone bool
	//priority bool
}

func SameOrder(currentOrder Order, orders []Order) bool {
	for _, i := range orders {
		if currentOrder.Pushed.Floor == i.Pushed.Floor {
			return true;
		}
	}
	return false;
}

/*func DeleteOrder(i, orders []Order) {
	//orders[i] = nil;
}*/

func Abs(x int) int {
	if x < 0 {
		return x*(-1)
	} else {
		return x
	}
}

func DistanceSquared(x int, y int) int {
	return (x-y)*(x-y)
}


//For network, use: func nearestFloor(ip network.IP, orders []Order, lastFloor int, currentFloor int, currentDirection elevio.MotorDirection) int
func NearestOrder(orders []Order, lastFloor int, currentDirection elevio.MotorDirection) Order {
	var nearestOrder Order
    nearestOrder.Pushed.Floor = -1
    nearestOrder.Pushed.Button = 2

	shortestDistance := -1
	
	var floorDistance int
	for _, i := range orders {
		
		/*if i.isDone != ip {
		continue
	}*/
		
		isOrderAbove := i.Pushed.Floor - lastFloor > 0
		isOrderBelow := i.Pushed.Floor - lastFloor < 0

		isTypeUp := i.Pushed.Button == elevio.BT_HallUp
		isTypeDown := i.Pushed.Button == elevio.BT_HallDown

		isGoingUp := currentDirection == elevio.MD_Up
		isGoingDown := currentDirection == elevio.MD_Down

		if (isOrderAbove && isGoingUp && !isTypeDown) || (isOrderBelow && isGoingDown && !isTypeUp) {
			var x int = (lastFloor - i.Pushed.Floor)
			floorDistance = Abs(x)
			//floorDistance = DistanceSquared(lastFloor, i.Pushed.Floor)
		}

		if (isOrderAbove || isTypeUp) && isGoingDown {
			var x int = (lastFloor + i.Pushed.Floor)
			floorDistance = Abs(x)
			//floorDistance = DistanceSquared(lastFloor, 0) + DistanceSquared(0, i.Pushed.Floor)
		}

		if (isOrderBelow || isTypeDown) && isGoingUp {
			var x int = (3 - lastFloor + 3 - i.Pushed.Floor)
			floorDistance = Abs(x)
			//floorDistance = DistanceSquared(lastFloor, 3) + DistanceSquared(4, i.Pushed.Floor)
		}

		if (nearestOrder.Pushed.Floor == -1 || shortestDistance > floorDistance) {
			nearestOrder = i
			shortestDistance = floorDistance
		}

	}
	if(shortestDistance==-1){
		nearestOrder.Pushed.Floor=lastFloor
	}

	return nearestOrder

}

