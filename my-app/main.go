package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Device represents a connected device with its ID, MAC address, and firmware version.
type Device struct {
	ID       int    `json:"id"`       // Unique identifier for the device
	Mac      string `json:"mac"`      // MAC address of the device
	Firmware string `json:"firmware"` // Firmware version of the device
}

// metrics defines the Prometheus metrics used in the application.
type metrics struct {
//The use of pointers (*) or non-pointers in the metrics struct depends on how the Prometheus client library implements these types:
//Non-pointer types (prometheus.Gauge, prometheus.Summary): Used for simple metrics.
//Pointer types (*prometheus.GaugeVec, *prometheus.CounterVec, etc.): Used for complex metrics that manage multiple labeled metrics

	devices       prometheus.Gauge         // Gauge to track the number of connected devices
	info          *prometheus.GaugeVec     // GaugeVec to track application version information
	upgrades      *prometheus.CounterVec   // CounterVec to track the number of device upgrades
	duration      *prometheus.HistogramVec // HistogramVec to track request durations
	loginDuration prometheus.Summary       // Summary to track login request durations
}

// NewMetrics initializes and registers Prometheus metrics.
func NewMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		devices: prometheus.NewGauge(prometheus.GaugeOpts{ //GaugeOpts is a struct in the Prometheus
			Namespace: "myapp",              // Metric namespace
			Name:      "connected_devices",  // Metric name
			Help:      "Number of currently connected devices.", // Metric description
		}),
		info: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "myapp",
			Name:      "info",
			Help:      "Information about the My App environment.",
		}, []string{"version"}), // Label for version information
		upgrades: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "myapp",
			Name:      "device_upgrade_total",
			Help:      "Number of upgraded devices.",
		}, []string{"type"}), // to findout how many of each device type has been upgraded
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "myapp",
			Name:      "request_duration_seconds",
			Help:      "Duration of the request.",
			Buckets:   []float64{0.1, 0.15, 0.2, 0.25, 0.3}, // Buckets for request duration
		}, []string{"status", "method"}), // Labels for status and method
		loginDuration: prometheus.NewSummary(prometheus.SummaryOpts{
			Namespace:  "myapp",
			Name:       "login_request_duration_seconds",
			Help:       "Duration of the login request.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}, // Quantiles for summary
		}),
	}
	reg.MustRegister(m.devices, m.info, m.upgrades, m.duration, m.loginDuration) // Register metrics
	return m
}

// Global variables
var dvs []Device // Slice to store connected devices
var version string // Application version

// init initializes the application with some default devices and version.
func init() {
	version = "2.10.5" // Set application version

	// Initialize with some default devices
	dvs = []Device{
		{1, "5F-33-CC-1F-43-82", "2.1.6"},
		{2, "EF-2B-C4-F5-D6-34", "2.1.6"},
	}
}

// main is the entry point of the application.
func main() {
	reg := prometheus.NewRegistry() // Create a new Prometheus registry
	m := NewMetrics(reg)            // Initialize metrics

	m.devices.Set(float64(len(dvs))) // Set the initial number of connected devices
	m.info.With(prometheus.Labels{"version": version}).Set(1) // Set version info metric
	// With is a method provided by prometheus.GaugeVec to select or create a specific labeled instance of the Gauge metric.
	// takes a prometheus.Labels map, where:The key is the label name ("version").
	// Set(1) sets the value of the Gauge metric to 1.


	// Create a new ServeMux for device-related endpoints
	dMux := http.NewServeMux()//servermux: hhtp request multiplexser:It allows you to define multiple routes (endpoints) in your application, each with its own handler.
	rdh := registerDevicesHandler{metrics: m} // Handler for device registration
	mdh := manageDevicesHandler{metrics: m}  // Handler for device management

	lh := loginHandler{}                     // Handler for login
	mlh := middleware(lh, m)                 // Wrap login handler with middleware

	dMux.Handle("/devices", rdh)             // Register devices endpoint
	dMux.Handle("/devices/", mdh)            // Register device management endpoint
	dMux.Handle("/login", mlh)               // Register login endpoint

	// Create a new ServeMux for Prometheus metrics
	pMux := http.NewServeMux()
	promHandler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	pMux.Handle("/metrics", promHandler)     // Register metrics endpoint

	// Start the HTTP server for device endpoints on port 8080
	go func() {
		log.Fatal(http.ListenAndServe(":8080", dMux))
	}()

	// Start the HTTP server for Prometheus metrics on port 8081
	go func() {
		log.Fatal(http.ListenAndServe(":8081", pMux))
	}()

	// Block forever to keep the application running
	select {}
}

// registerDevicesHandler handles device registration and listing.
type registerDevicesHandler struct {
	metrics *metrics
}

// first this below func has its own name but beacuse of its functionality that it serve http requests
// we use servehttp that is predifed interface of http requests and it is our interface that is defined in http lib so we didnt need to 
//define it as interface 
// ServeHTTP handles HTTP requests for device registration and listing.
//so serverhttp is our interface and in our functions that inhertince it we should tell it which struct will use in this interface

func (rdh registerDevicesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		getDevices(w, r, rdh.metrics) // Handle GET request to list devices
	case "POST":
		createDevice(w, r, rdh.metrics) // Handle POST request to create a device
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// getDevices returns a list of all connected devices.
func getDevices(w http.ResponseWriter, r *http.Request, m *metrics) {
	now := time.Now() // Start timing the request

	b, err := json.Marshal(dvs) // Convert devices to JSON
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sleep(200) // Simulate some processing time

	// Record request duration in Prometheus
	m.duration.With(prometheus.Labels{"method": "GET", "status": "200"}).Observe(time.Since(now).Seconds())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(b) // Send JSON response
}

// createDevice adds a new device to the list of connected devices.
// curl -d "{\"id\":4,\"mac\":\"5F-34-CC-1F-43-82\",\"firmware\":\"4.1.6\"}" localhost:8080/devices

func createDevice(w http.ResponseWriter, r *http.Request, m *metrics) {
// json.NewDecoder(r.Body): Creates a new JSON decoder that reads from the request body.
// Decode(&dv): Attempts to decode the JSON data from the request bodyThe & is important here; it passes a pointer to dv, so the Decode function can modify the dv variable directly.
// dvs = append(dvs, dv):Important: This code is not thread-safe! If multiple requests try to create devices concurrently, you'll have race conditions and data corruption. You'll need to use a mutex to protect access to the dvs slice.

	var dv Device

	err := json.NewDecoder(r.Body).Decode(&dv) // Decode JSON request body into a Device struct
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dvs = append(dvs, dv) // Add the new device to the list

	m.devices.Set(float64(len(dvs))) // Update the connected devices metric

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Device created!")) // Send success response
}

// upgradeDevice upgrades the firmware of a specific device.
func upgradeDevice(w http.ResponseWriter, r *http.Request, m *metrics) {
	path := strings.TrimPrefix(r.URL.Path, "/devices/") // Extract device ID from URL path

	id, err := strconv.Atoi(path) // Convert ID to integer
	if err != nil || id < 1 {
		http.NotFound(w, r)
	}

	var dv Device
	err = json.NewDecoder(r.Body).Decode(&dv) // Decode JSON request body into a Device struct
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Find the device by ID and update its firmware
	for i := range dvs {
		if dvs[i].ID == id {
			dvs[i].Firmware = dv.Firmware
		}
	}
	sleep(1000) // Simulate some processing time

	m.upgrades.With(prometheus.Labels{"type": "router"}).Inc() // Increment upgrade counter

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Upgrading...")) // Send response
}

// manageDevicesHandler handles device management requests.
type manageDevicesHandler struct {
	metrics *metrics
}

// ServeHTTP handles HTTP requests for device management.
func (mdh manageDevicesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "PUT":
		upgradeDevice(w, r, mdh.metrics) // Handle PUT request to upgrade a device
	default:
		w.Header().Set("Allow", "PUT")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// sleep simulates processing time by sleeping for a random duration.
func sleep(ms int) {
	rand.Seed(time.Now().UnixNano())
	now := time.Now()
	n := rand.Intn(ms + now.Second())
	time.Sleep(time.Duration(n) * time.Millisecond)
}

// loginHandler handles login requests.
type loginHandler struct{}

// ServeHTTP handles HTTP requests for login.
func (l loginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sleep(200) // Simulate some processing time
	w.Write([]byte("Welcome to the app!")) // Send response
}

// middleware wraps an HTTP handler to measure request duration.
func middleware(next http.Handler, m *metrics) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now() // Start timing the request
		next.ServeHTTP(w, r)
		m.loginDuration.Observe(time.Since(now).Seconds()) // Record login duration
	})
}