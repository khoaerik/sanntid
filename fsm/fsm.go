package fsm

import "../elevio"
import "../queue"
import "../messages"
import "fmt"
import "time"
import "flag"
import "os/exec"
import "os"
import "encoding/json"
import "../network/localip"
import "../network/peers"
import "../network/bcast"
import "../rwfile"
import "sync"

var port int = 15678



func Fsm(){
    var state string
    var id string
    numFloors := 4
    var hallOrders [4][2]bool
    var cabRequests [4]bool
    orders :=[]queue.Order{}
    elevio.Init("localhost:15657", numFloors)

    flag.StringVar(&id, "id", "", "id of this peer")
    flag.Parse()//?

    if id == "" {
    	localIP, err := localip.LocalIP()
    	if err != nil {
    		fmt.Println(err)
    		localIP = "DISCONNECTED"
    	}
    	id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
    }

    
    status := new(messages.StatusStruct)                         
    status.HallRequests = make([][2]bool, 4)
    status.States = make(map[string]*messages.StateValues)

    test := new(messages.StatusStruct)
    test.HallRequests = make([][2]bool, 4)
    test.States = make(map[string]*messages.StateValues)

   
    var dirn elevio.MotorDirection

    
    statusMsg := new(messages.StatusMsg)
    statusMsg.SenderId = id
    statusMsg.Status = *status

    orderMsg := new(messages.OrderMsg)
    orderMsg.SenderId = id

    ackMsg := new(messages.AckMsg)
    ackMsg.SenderId = id
    ackMsg.Ack = false
    
    drv_buttons := make(chan elevio.ButtonEvent)
    drv_floors  := make(chan int)
    drv_obstr   := make(chan bool)
    drv_stop    := make(chan bool)

    updatePeer := make(chan peers.PeerUpdate)
    enablePeerTransmitter := make(chan bool)

    ch := messages.Channels {
        ElevStatusTxCh: make(chan messages.StatusMsg, 100),
        ElevStatusRxCh: make(chan messages.StatusMsg, 100),
        HallRequestTxCh: make(chan messages.OrderMsg, 100),
        HallRequestRxCh: make(chan messages.OrderMsg, 100),
        AckTxCh: make(chan messages.AckMsg, 100),
        AckRxCh: make(chan messages.AckMsg, 100),
    }


    timer := time.NewTimer(3*time.Second)
    timer.Stop()

    watchDog := time.NewTimer(4*time.Second)
    watchDog.Stop()


    var lastFloor int
    var direction string

    var res map[string][][]bool
    var result map[string][][]bool

    var peersList peers.PeerUpdate

    var lastOrder queue.Order
    lastOrder = structInit(lastOrder)
    var nearestOrder queue.Order
    nearestOrder = structInit(nearestOrder)
    

    go elevio.PollButtons(drv_buttons)
    go elevio.PollFloorSensor(drv_floors)
    go elevio.PollObstructionSwitch(drv_obstr)
    go elevio.PollStopButton(drv_stop)

    go peers.Transmitter(15649, id, enablePeerTransmitter)
    go peers.Receiver(15649, updatePeer)


 
    go bcast.Transmitter(port, ch.ElevStatusTxCh, ch.HallRequestTxCh, ch.AckTxCh)
    go bcast.Receiver(port, ch.ElevStatusRxCh, ch.HallRequestRxCh, ch.AckRxCh)
    
  
    
    var mutex = &sync.Mutex{}

    statusMsg.SenderId = id
    ackMsg.SenderId = id

    var filename string = "status.txt"

    //---------------- INIT ----------------_//
    dirn = elevio.MD_Down//init
    elevio.SetMotorDirection(dirn)//init
    lastFloor=0//init
    state = "initState"//init

    var intCabRequests [4]int
    intCabRequests = rwfile.ReadFromFile(filename)
    var fileOrder queue.Order
    for i, _ := range intCabRequests {
        if intCabRequests[i] == 0 {
            cabRequests[i] = false
        } else {
            cabRequests[i] = true
            fileOrder.Pushed.Button=2
            fileOrder.Pushed.Floor=i
            orders = append(orders, fileOrder)
        }
    }


    fmt.Println(cabRequests)




    for {
        
        /*if lastOrder.Pushed.Floor == lastFloor {
            watchDog.Reset(4*time.Second)
        }*/
        

        fmt.Printf("ORDERS: %v\n",orders)
        nearestOrder=queue.NearestOrder(orders, lastFloor, dirn)
    	
        mutex.Lock()
        updateStatusStruct(id, status, lastFloor, direction, cabRequests, state)
        mutex.Unlock()

        statusMsg.Status = *status
        fmt.Sprintf("original %v", statusMsg.Status)
        fmt.Sprintf("%v",res)

        
        select {


        case ElevStatus := <- ch.ElevStatusRxCh:
            var dummy messages.StatusMsg
            dummy = ElevStatus
            mutex.Lock()
            statusMsg.Status.States[ElevStatus.SenderId] = dummy.Status.States[ElevStatus.SenderId]
            mutex.Unlock()

        case HallRequest := <- ch.HallRequestRxCh:
            if id == HallRequest.TakerId {
                        
                elevio.SetButtonLamp(HallRequest.Button.Button, HallRequest.Button.Floor, true)

                var newOrder queue.Order
                newOrder.Pushed.Floor = HallRequest.Button.Floor
                newOrder.Pushed.Button = HallRequest.Button.Button
                cabRequests[HallRequest.Button.Floor]=true
                rwfile.WriteToFile(cabRequests, filename)

                if !queue.SameOrder(newOrder, orders) && newOrder!=lastOrder{
                    fmt.Println("added")
                    orders = append(orders, newOrder)

                }

                        
            }


            switch state {
                case "moving":
                case "doorOpen":
                case "idle":

                    

                    if nearestOrder.Pushed.Floor==lastFloor {

                        elevio.SetButtonLamp(lastOrder.Pushed.Button, lastOrder.Pushed.Floor, false)
                        elevio.SetButtonLamp(2, lastOrder.Pushed.Floor, false)
                        fmt.Println("LAST ORDER: ", lastOrder)
                        lastOrder.Pushed.Button = HallRequest.Button.Button
                        orders=remove(orders,lastOrder)
                        cabRequests[lastFloor] = false
                        rwfile.WriteToFile(cabRequests, filename)

                        if lastOrder.Pushed.Button == 0 || lastOrder.Pushed.Button == 1 {
                            ackMsg.Ack = true
                            ackMsg.Button.Floor = lastOrder.Pushed.Floor
                            ackMsg.Button.Button = lastOrder.Pushed.Button
                            for i :=0; i<5; i++{
                                ch.AckTxCh <- *ackMsg
                            }
                        }

                        timer.Reset(3*time.Second)

                        elevio.SetDoorOpenLamp(true)
                        elevio.SetMotorDirection(elevio.MD_Stop)
                        state = "doorOpen"
                     
                    } else {
                        dirn=chooseDir(nearestOrder.Pushed.Floor,lastFloor)
                        direction=chooseStringDir(nearestOrder.Pushed.Floor,lastFloor)
                        elevio.SetMotorDirection(dirn)
                        state="moving"

                    }
                    for i :=0; i<5; i++{
                        ch.ElevStatusTxCh <- *statusMsg
                    }
                    watchDog.Reset(15*time.Second)
            }



        case Ack := <- ch.AckRxCh:
            if Ack.Ack == true {
                    //cabRequests[Ack.Button.Floor] = false
                    //rwfile.WriteToFile(cabRequests, filename)
                    statusMsg.Status.HallRequests[Ack.Button.Floor][Ack.Button.Button]=false
                }


        case p := <- updatePeer:
        	fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", p.Peers)
			fmt.Printf("  New:      %q\n", p.New)
			fmt.Printf("  Lost:     %q\n", p.Lost)

			peersList.Peers = p.Peers


        case targetFloor := <- drv_buttons:



            if targetFloor.Button == 0 {
                hallOrders[targetFloor.Floor][0] = true
                status.HallRequests[targetFloor.Floor][0] = true
       
                
                result = Cost(*status)
                fmt.Println(result)
                
                orderMsg.Button = targetFloor
                orderMsg.SenderId = id

                for takerId, element := range result {
                    if element[targetFloor.Floor][targetFloor.Button] == true {
                        fmt.Println(element[targetFloor.Floor][0], element[targetFloor.Floor][1])
                        orderMsg.TakerId = takerId
                        fmt.Println(takerId,orderMsg.Button)
                        for i :=0; i<10; i++{
                            ch.HallRequestTxCh <- *orderMsg
                        }
                    }
                
                }

            }

            if targetFloor.Button == 1 {
                hallOrders[targetFloor.Floor][1] = true
                status.HallRequests[targetFloor.Floor][1] = true

                result = Cost(*status)
                fmt.Println(result)

                orderMsg.Button = targetFloor
                orderMsg.SenderId = id
                for takerId, element := range result {
                    if element[targetFloor.Floor][targetFloor.Button] == true {
                        fmt.Println(element[targetFloor.Floor][0], element[targetFloor.Floor][1])
                        orderMsg.TakerId = takerId
                        fmt.Println(takerId,orderMsg.Button)
                        for i :=0; i<10; i++{
                            ch.HallRequestTxCh <- *orderMsg
                        }
                    }
                
                }


            }


            var order queue.Order;
            order.Pushed.Floor = targetFloor.Floor
            order.Pushed.Button = targetFloor.Button
            if !queue.SameOrder(order, orders) {
                if (order.Pushed.Button == 2) {
                	orders = append(orders, order)
                    elevio.SetButtonLamp(targetFloor.Button, targetFloor.Floor, true)
                    cabRequests[order.Pushed.Floor]=true
                    rwfile.WriteToFile(cabRequests, filename)
                } 
            }
            for i :=0; i<5; i++{
                ch.ElevStatusTxCh <- *statusMsg
            }


            switch state {
        		case "moving":
        		case "doorOpen":
        		case "idle":
                    
        			if nearestOrder.Pushed.Floor==lastFloor {
                        
                        elevio.SetButtonLamp(lastOrder.Pushed.Button, lastOrder.Pushed.Floor, false)
                        elevio.SetButtonLamp(2, lastOrder.Pushed.Floor, false)
                        fmt.Println("LAST ORDER: ", lastOrder)
                        lastOrder.Pushed.Button = targetFloor.Button
                        orders=remove(orders,lastOrder)
                        cabRequests[lastFloor] = false
                        rwfile.WriteToFile(cabRequests, filename)
                        fmt.Println("Removenow2")

                        if lastOrder.Pushed.Button == 0 || lastOrder.Pushed.Button == 1 {
                            ackMsg.Ack = true
                            ackMsg.Button.Floor = lastOrder.Pushed.Floor
                            ackMsg.Button.Button = lastOrder.Pushed.Button
                            for i :=0; i<5; i++{
                                ch.AckTxCh <- *ackMsg
                            }
                        }
        				timer.Reset(3*time.Second)
						elevio.SetDoorOpenLamp(true)
						elevio.SetMotorDirection(elevio.MD_Stop)
						state = "doorOpen"
     				
        			} else {
                        dirn=chooseDir(nearestOrder.Pushed.Floor,lastFloor)
                        direction=chooseStringDir(nearestOrder.Pushed.Floor,lastFloor)
                        elevio.SetMotorDirection(dirn)
                        state="moving"

                    }
                    for i :=0; i<5; i++{
                        ch.ElevStatusTxCh <- *statusMsg
                    }
        	}



        case currentFloor := <- drv_floors:
            watchDog.Stop()
      		lastFloor = currentFloor
            lastOrder = nearestOrder
            nearestOrder=queue.NearestOrder(orders, lastFloor, dirn)

            
            

            switch state {
            	case "initState":
                    

            		if (currentFloor==0){
                        dirn=elevio.MD_Stop
                        direction = "stop"
                      
            			elevio.SetMotorDirection(dirn)
            			state = "idle"
                       
                        lastFloor=0
                        lastOrder.Pushed.Floor=0


                        if len(orders) > 0 {
                        dirn = elevio.MD_Up
                        direction = "up"
                        elevio.SetMotorDirection(dirn)
                        state = "moving"
                    }


            		}


                    for i :=0; i<5; i++{
                                ch.ElevStatusTxCh <- *statusMsg
                    }

            	case "idle":



            	case "doorOpen":

                case "motorStop":
                    enablePeerTransmitter <- true
                    state = "moving"

            	case "moving":
            		if currentFloor == numFloors-1 {
            			dirn = elevio.MD_Down
                        direction = "down"
                        
            		} 
           			if currentFloor == 0 {
           				dirn = elevio.MD_Up
                        direction = "up"
                        
            		}

            		elevio.SetMotorDirection(dirn)

            		if lastOrder.Pushed.Floor == lastFloor{

                        fmt.Println("REMOVE NOW")

                        elevio.SetDoorOpenLamp(true)
                        elevio.SetMotorDirection(elevio.MD_Stop)

                        fmt.Println("LAST ORDER: ", lastOrder)

                        orders=remove(orders,lastOrder)
                        cabRequests[lastOrder.Pushed.Floor] = false
                        rwfile.WriteToFile(cabRequests, filename)
                        
                        
                        if lastOrder.Pushed.Button == 0 || lastOrder.Pushed.Button == 1 {
                            ackMsg.Ack = true
                            ackMsg.Button.Floor = lastOrder.Pushed.Floor
                            ackMsg.Button.Button = lastOrder.Pushed.Button
                            for i :=0; i<5; i++{
                                ch.AckTxCh <- *ackMsg
                            }
                        }


                        timer.Reset(3*time.Second)
                        state = "doorOpen" 
                        
                    } else {
                        state="moving"
                        watchDog.Reset(4*time.Second)
        				dirn=chooseDir(nearestOrder.Pushed.Floor,lastFloor)
                        direction = chooseStringDir(nearestOrder.Pushed.Floor,lastFloor)
                        elevio.SetMotorDirection(dirn)
                        
                    }
                    for i :=0; i<5; i++{
                        ch.ElevStatusTxCh <- *statusMsg
                    }
                    
                   
        		}

		        	

        case <- timer.C:
    		switch state {
	    		case "idle":
	    		case "moving":
	    		case "doorOpen":
					elevio.SetDoorOpenLamp(false)
                    fmt.Println("TIMER!")
                    
                    orders=remove(orders,lastOrder)
                    cabRequests[lastOrder.Pushed.Floor] = false
                    rwfile.WriteToFile(cabRequests, filename)


                    if dirn == elevio.MD_Up {
                        elevio.SetButtonLamp(0, lastOrder.Pushed.Floor, false)
                    }
                    if dirn == elevio.MD_Down {
                        elevio.SetButtonLamp(1, lastOrder.Pushed.Floor, false)
                    }
                    elevio.SetButtonLamp(lastOrder.Pushed.Button, lastOrder.Pushed.Floor, false)
                    nearestOrder=queue.NearestOrder(orders, lastFloor, dirn)
                    if (len(orders)>0 && lastOrder!=nearestOrder){
                        dirn=chooseDir(nearestOrder.Pushed.Floor,lastFloor)
                        direction = chooseStringDir(nearestOrder.Pushed.Floor,lastFloor)
                        elevio.SetMotorDirection(dirn)
                   
                        state = "moving"
                        
                    }else{
                        state = "idle"
                        
                    }
                    for i :=0; i<5; i++{
                        ch.ElevStatusTxCh <- *statusMsg
                    }
    		}
        
        case <- watchDog.C:
            fmt.Println("WATCHDOG TIMED OUT")
            enablePeerTransmitter <- false
            state = "motorStop"
            /*if motorstopp {
            under state "moving"
            enablePeerTransmitter <- false
        }*/ // HUSK Å HA MULIGHET FOR Å KOMME TILBAKE!!! enablePeerTransmitter <- true
		
            
        case obstr := <- drv_obstr:
            fmt.Printf("%+v\n", obstr)
            if obstr {
                elevio.SetMotorDirection(elevio.MD_Stop)
            } else {
                elevio.SetMotorDirection(dirn)
            }
            
        case stop := <- drv_stop:
            fmt.Printf("%+v\n", stop)
            for f := 0; f < numFloors; f++ {
                for b := elevio.ButtonType(0); b < 3; b++ {
                    elevio.SetButtonLamp(b, f, false)
                }
            }
        }

    }    
}




func initelev(dirn elevio.MotorDirection, state string, lastFloor int){
    dirn = elevio.MD_Down//init
    elevio.SetMotorDirection(dirn)//init
    lastFloor=0
}


func updateStatusStruct(id string, status *messages.StatusStruct, lastFloor int, direction string, cabRequests [4]bool, state string){
        status.States[id] = new(messages.StateValues)
        status.States[id].Floor = lastFloor
        status.States[id].Direction = direction
        status.States[id].CabRequests = cabRequests
        status.States[id].Behaviour = state

}


func remove(orders []queue.Order, order queue.Order) []queue.Order {
        for j, i := range orders {
            if order == i {
                orders[len(orders)-1], orders[j] = orders[j], orders[len(orders)-1]
                return orders[:len(orders)-1]
            }
        }
        return orders
}

func structInit(order queue.Order) queue.Order {
    order.Pushed.Floor = 0
    order.Pushed.Button = 0
    return order
}

func chooseDir(goalfloor int, lfloor int) elevio.MotorDirection{
    var dirn elevio.MotorDirection
    if goalfloor > lfloor {
        dirn = elevio.MD_Up
    }else if goalfloor < lfloor{
        dirn = elevio.MD_Down
    }else {
        dirn =elevio.MD_Stop
    }
    return dirn
}
    
func chooseStringDir(goalfloor int, lfloor int) string {
    var direction string
    if goalfloor > lfloor {
        direction = "up"
    }else if goalfloor < lfloor{
        direction = "down"
    }else {
        direction = "stop"
    }
    return direction
}



func Cost(status messages.StatusStruct) map[string][][]bool {
        
        test, err := json.Marshal(status)
        if err != nil {
            fmt.Println("error:", err)
        }

        result, err := exec.Command("sh", "-c", "./hall_request_assigner --input '"+string(test)+"'").Output()

        if err != nil {
            fmt.Println("error:", err)
        }
        resultOrders := new(map[string][][]bool)
        json.Unmarshal(result, resultOrders)
        
        
        return *resultOrders
}