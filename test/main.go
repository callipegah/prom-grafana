package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	// "github.com/uber/jaeger-lib/metrics/prometheus"
)

type Device struct { //define struct for device
	ID       int    `json:"id"`
	Mac      string `json:"mac"`
	Firmware string `json:"firemare"`
}

var dvs []Device   // define slices for device
var version string //declare a global version variable
///typicllay it used for defing global variables

func init() {
	version = "2.10"
	dvs = []Device{
		{1, "5f-4c", "2.1.6"}, //put values in slices of device
		{2, "5f-4c", "2.1.7"},
		{3, "5f-4c", "2.1.7"},
	}
}

func main() {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	m.devices.Set(float64(len(dvs)))
	m.info.With(prometheus.Labels{"version": version}).Set(1) //use constant value of 1
 
	//define manual prom handler
	promHandler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg})

	//define server with 8081 port
	// http.Handle("/metrics",promhttp.Handler())//define endpoint and tell it wich func will be execute
	http.Handle("/metrics", promHandler)
	http.HandleFunc("/devices", getDevices) //define endpoint and its func and dont put params to that func
	http.ListenAndServe(":8081", nil)       //it should write in the last line

	//using go routines to create multiple servers by using goroutines
	//by creating seprate http request multiplexers
	dMux := http.NewServeMux() //we use this to serve the main content
	rdh:=registerDeviceHandler(metrics:m)
	mdh:=manageDeviceHandler(metrics:m)
	dMux.HandleFunc("/devices", rdh)
	dMux.HandleFunc("/devices/", mdh)

	pMux := http.NewServeMux()
	promhandler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	pMux.Handle("/metrics", promhandler) //this one is for prometheus

	go func() {
		log.Fatal(http.ListenAndServe(":8080", dMux))
	}()

	go func() {
		log.Fatal(http.ListenAndServe(":8081", pMux))
	}()

	//to perevent main function from exiting //that blocks until our goroutine is running
	select {}
}

func getDevices(w http.ResponseWriter, r *http.Request) { //it is func  that will get http  inputs so it get writer and reader
	b, err := json.Marshal(dvs) //Marshal returns the JSON encoding of dvs
	print("b values is :", b)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(b) //write data to connction

}

type metrics struct {
	devices prometheus.Gauge
	info    *prometheus.GaugeVec /// for expose the version of running app
	//for setting version label with the actual version of application
    duration *prometheus.HistogramVec
}

func NewMetrics(reg prometheus.Registerer) *metrics {
	//return the pointer to the matrix struct
	m := &metrics{
		devices: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "myapp",
			Name:      "connected_devices",
			Help:      "Number of currently connected devices.",
		}),
		info: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "myapp",
			Name:      "info",
			Help:      "Information about the My App environment.",
		},
			[]string{"version"}),
		upgrades : prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace:"myapp",
			Name:"device_upgrade_total",
			Help: "Number of upgraded"
		},[]string{"type"}),// counter param of prometheus
	},
	duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace : "myapp",
		Name :"request",
		Help: "mio",
		Buckets:[]float64{0.1,0.15},

	},[]string{"status","method"}),
	reg.MustRegister(m.devices, m.info,m.upgrades) //register it by prometheus registry
	return m                            //return a pointer

}

//u can use built-in collerctor to register it with custom primitos register
//



func createDevice(w http.ResponseWriter,r *http.Request){
	var dv Device

	err:=json.NewDecoder(r.Body).Decode(&dv)
	if err!=nil{
		http.Error(w,err.Error(),http.StatusBadRequest)
		return
	}

	m.devices.Set(float64(len(dvs)))

	dvs=append(dvs, dv)
	w.WriteHeader(http.StatusCreated)//set the staus=201
	w.Write([]byte("Device created"))
}

func (rdh registerDeviceHandler)ServerHTTP(w http.ResponseWriter,r *http.Request,m *metrics){
	

	switch r.Method {
	case "GET":
		getDevices(w,r)
	case "POST":
		createDevice(w,r,rdh.)

	default:
		w.Header().Set("Allow","GET , POST")
		http.Error(w,"not allowed",http.StatusMethodNotAllowed)

	}

}

type registerDeviceHandler struct{
	metrics *metrics//we can increase the device count
}


func upgradeDevice(w http.ResponseWriter,r *http.Request,m *metrics){
	path:=strings.TrimPrefix(r.URL.Path,"/devices/")//to get id of the device we trim the path
	
	id,err:=strconv.Atoi(path)//convert string id to integer
    var dv Device 
	err=json.NewDecoder(r.Body).Decode(&dv)

	if err !=nil{
		http.Error(w,err.Error(),http.StatusBadRequest)
		return

	}

	for i:=range dvs{
		if dvs[i].ID==id{
			dvs[i].firemare=dv.firemare
		}
	}

	m.upgrades.With(prometheus.Labels{"type":"router"}).Inc()

	w.WriterHeader(http.StatusAccepted)
	w.Write([]byte("upgrading ..."))
}


type manageDeviceHandler struct{
	metrics *metrics
}


func (mdh manageDeviceHandler)ServerHTTP(w http.ResponseWriter,r *http.Request){
	switch r.Method{
	case "PUT":
		upgradeDevice(w,r,mdh.metrics)
	}
    default:{
		w.Header().Set("Allow","PUT")
		http.Error(w,"method not allowed",http.StatusMethodNotAllowed)
	}

}


func sleep(ms int){
	rand.Seed(time.Now().UnixNano())
	now :=time.Now()
	n:=rand.Intn(ms+ now.Second())
	time.Sleep(time.Duration(n)*time.Millisecond)
}