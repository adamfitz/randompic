package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

//go:embed static/index.html
var staticIndexFile string

var (
	randomImage   string
	imageMutex    sync.Mutex         // To ensure thread-safe access to `randomImage`
	IndexTemplate *template.Template // capitalised to allow "export" and usage in init funcion
	/*
		embed package includes the index file contents as a string but the template engine expects a file path.  Instead parse the string content instead of trying to use a filepath
	*/
)

// Config represents the configuration structure for exclusions
type Config struct {
	ExcludedExtensions  []string `json:"excludedExtensions"`
	ExcludedDirectories []string `json:"excludedDirectories"`
	ImageDirectory      string   `json:"imageDirectory"`
	DisplaySeconds      int      `json:"displaySeconds"`
}

func init() {
	// Configure lumberjack logger for log rotation
	log.SetOutput(&lumberjack.Logger{
		Filename:   "./randompic.log", // Log file name
		MaxSize:    10,                // Maximum size in megabytes before it rotates
		MaxBackups: 5,                 // Maximum number of old log files to keep
		MaxAge:     0,                 // Maximum number of days to retain old logs (0 means no limit)
		Compress:   false,             // Do not compress log files
	})

	// parse the embedded index.html string to create a new template "file"
	var tmplErr error
	IndexTemplate, tmplErr = template.New("index").Parse(staticIndexFile)
	if tmplErr != nil {
		log.Fatalf("Error parsing template: %v", tmplErr)
	}
}

// loadConfig reads the exclusion configuration from a JSON file
func loadConfig(configPath string) (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// ListFiles recursively traverses a directory and its subdirectories,
// returning a slice of absolute file paths for all files.
func ListFiles(root string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// If it's not a directory, add the file path to the slice
		if !d.IsDir() {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			files = append(files, absPath)
		}
		return nil
	})

	return files, err
}

// SelectRandomElement selects a random element from a slice of strings.
func SelectRandomElement(elements []string) (string, error) {
	if len(elements) == 0 {
		return "", fmt.Errorf("the list is empty")
	}

	// Create a new random source and generator
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Generate a random index
	randomIndex := r.Intn(len(elements))

	// Return the random element
	return elements[randomIndex], nil
}

func pageHandler(w http.ResponseWriter, r *http.Request) {
	/*
		Receives the absolute location of an image file and renders it on the page.
	*/

	// load config file to get the timeout value
	configPath := filepath.Join(".", "config.json")
	config, err := loadConfig(configPath)
	if err != nil {
		http.Error(w, "Error loading config: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Error loading config: %v", err)
		return
	}

	// Parse the embedded template content once during initialization
	tmplParsed, err := template.New("index").Parse(staticIndexFile)
	if err != nil {
		http.Error(w, "Error parsing template: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Error parsing template: %v", err)
		return
	}

	// Safely access the randomImage variable
	image := func() string {
		imageMutex.Lock()
		defer imageMutex.Unlock()
		// Strip the base directory and return a relative path
		// Assuming randomImage is the absolute path, so remove the provided path loaded from the configuratoin file
		return "/images" + randomImage[len(config.ImageDirectory):]
	}()

	// Render the template with image data and timeout value
	data := struct {
		ImageURL       string
		DisplaySeconds int
	}{
		ImageURL:       image,
		DisplaySeconds: config.DisplaySeconds, // number of seconds to display an image pulled from the config file
	}
	if err := tmplParsed.Execute(w, data); err != nil {
		http.Error(w, "Error rendering template: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Error executing template: %v", err)
	}
}

// loadAllImages loads all images from a directory while applying exclusions
func loadAllImages() []string {
	/*
		Load all images once and return a string slice with the absolute location of all read images,
		excluding certain files based on extension or directory name substring.
	*/

	// Load configuration
	configPath := filepath.Join(".", "config.json")
	config, err := loadConfig(configPath)
	if err != nil {
		log.Printf("Failed to load configuration: %v", err)
		return []string{} // Return an empty slice if config loading fails
	}

	// Get the list of files
	files, err := ListFiles(config.ImageDirectory)
	if err != nil {
		log.Println("Error:", err)
		return []string{} // Return an empty slice instead of nil
	}

	// Filtered list of files
	var filteredFiles []string

	// Loop through all the files and exclude those that match the conditions
	for _, file := range files {
		// Check if the file has an excluded extension
		ext := strings.ToLower(filepath.Ext(file))
		if contains(config.ExcludedExtensions, ext) {
			continue
		}

		// Check if the file starts with a dot (hidden files)
		if strings.HasPrefix(filepath.Base(file), ".") {
			continue
		}

		// Check if the file is in an excluded directory
		excluded := false
		for _, dirSubstring := range config.ExcludedDirectories {
			if strings.Contains(filepath.Dir(file), dirSubstring) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		// Add the file to the filtered list if it passes all conditions
		filteredFiles = append(filteredFiles, file)
	}

	return filteredFiles
}

// Helper function to check if a slice contains a string (used to filter file extensions and prefixes from the filteredFiles list)
func contains(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

func selectRandomImage(fileList []string) string {

	// Select a random element
	image, err := SelectRandomElement(fileList)
	if err != nil {
		log.Println("Error:", err)
		return ""
	}
	return image

}

func updateImagePeriodically(fileList []string, interval time.Duration) {
	for {
		// Select a new random image
		newImage := selectRandomImage(fileList)
		log.Printf("Displaying image: %s", newImage)

		// Update the shared randomImage variable safely
		imageMutex.Lock()
		randomImage = newImage
		imageMutex.Unlock()

		// Sleep for the specified interval
		time.Sleep(interval)
	}
}

func main() {

	start := time.Now() // time the loading of images
	// get the list of files (only runs once)
	fileList := loadAllImages()
	elapsed := time.Since(start)
	log.Printf("Loading fileList from disk took: %s", elapsed)

	// load config file
	configPath := filepath.Join(".", "config.json")
	config, _ := loadConfig(configPath)

	// Start the image updater in a goroutine
	go updateImagePeriodically(fileList, time.Duration(config.DisplaySeconds)*time.Second)

	// Serve images from the directory
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir(config.ImageDirectory))))

	// Serve the page
	http.HandleFunc("/", pageHandler)
	log.Println("Starting server on :80")
	log.Fatal(http.ListenAndServe(":80", nil))

}
