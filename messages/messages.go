package messages


//import "encoding/json"
//import "../network"
import "../elevio"


type StatusStruct struct {
	HallRequests [][2]bool `json:"hallRequests"`
	States map[string]*StateValues `json:"states"`
	/*Behaviour   string `json:"behaviour"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests [4]bool `json:"cabRequests"`//key kan be changed to int if more practical but remember to cast to string before JSON encoding!*/
}

type StateValues struct {
	Behaviour   string `json:"behaviour"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests [4]bool `json:"cabRequests"`//key kan be changed to int if more practical but remember to cast to string before JSON encoding!
}


type StatusMsg struct {
	SenderId string
	Status StatusStruct
}

type OrderMsg struct {
	SenderId string
	TakerId string
	Button elevio.ButtonEvent
}

type AckMsg struct {
	SenderId string
	Ack bool
	Button elevio.ButtonEvent
}

type Channels struct {
	ElevStatusTxCh chan StatusMsg
	ElevStatusRxCh chan StatusMsg
	HallRequestTxCh chan OrderMsg
	HallRequestRxCh chan OrderMsg
	AckTxCh chan AckMsg
	AckRxCh chan AckMsg
}

/*
func myRoutine(){
	for{
		select{
			<- time.After(time.Millisecond * 100):
				sync()
				chan <- msg
	}
	}

}

// res vil returnere hvem som skal gå dit, men vi trenger KUN id-en til den som skal gå dit


func sync(){

	// Send all the data on the network
	network <- msg
}

*/