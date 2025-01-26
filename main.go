package main

import (
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
)

var (
	randomImage       string
	imageMutex        sync.Mutex // To ensure thread-safe access to `randomImage`
	indexTemplatePath = "./index.html"
)

func init() {
	logFile, err := os.OpenFile("./randompic.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	log.SetOutput(logFile)
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

	// Load the template from disk
	tmplParsed, err := template.ParseFiles(indexTemplatePath)
	if err != nil {
		http.Error(w, "Error loading template: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Error parsing template: %v", err)
		return
	}

	// Safely access the randomImage variable
	image := func() string {
		imageMutex.Lock()
		defer imageMutex.Unlock()
		// Strip the base directory and return a relative path
		// Assuming randomImage is the absolute path, so remove "/mnt/photos"
		return "/images" + randomImage[len("/mnt/photos"):]
	}()

	// Render the template with image data
	if err := tmplParsed.Execute(w, struct{ ImageURL string }{ImageURL: image}); err != nil {
		http.Error(w, "Error rendering template: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Error executing template: %v", err)
	}
}

func loadAllImages(imageDirectory string) []string {
	/*
		Load all images once and returns a string slice with the absolute location of all read images,
		excluding certain files based on extension or prefix.
	*/

	// Setup the root directory
	rootDir := imageDirectory
	files, err := ListFiles(rootDir)
	if err != nil {
		log.Println("Error:", err)
		return []string{} // Return an empty slice instead of nil
	}

	// List of file extensions to exclude
	excludedExtensions := []string{".mp4", ".mov", ".heic"}

	// Filtered list of files
	var filteredFiles []string

	// Loop through all the files and exclude the ones that match the conditions
	for _, file := range files {
		// Check if the file has an excluded extension
		ext := strings.ToLower(filepath.Ext(file))
		if contains(excludedExtensions, ext) {
			continue
		}

		// Check if the file starts with a dot (hidden files)
		if strings.HasPrefix(filepath.Base(file), ".") {
			continue
		}

		// Add the file to the filtered list if it passes both conditions
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

	// get the list of files (only runs once)
	fileList := loadAllImages("/mnt/photos")

	// Start the image updater in a goroutine
	go updateImagePeriodically(fileList, 10*time.Second)

	// Serve images from the directory
	imageDirectory := "/mnt/photos"
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir(imageDirectory))))

	// Serve the page
	http.HandleFunc("/", pageHandler)
	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

}
