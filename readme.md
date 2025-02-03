# randompic

A basic go app that loads images from a directory and displays the image on a web page.

The app takes a local directory path as input, cleans up the file path (strips the path and starts a file server to serve images) and selects an image at random to display.

A new random image is selected to be displayed and the page automatically updated based on the integer value provided in the configuration file.

Photos are served from a directory on server/computer where the app runs, alternatively images could be mounted from a network share on the computer where the app is run from.

This page can be opened/displayed on an old tablet/device so it can be repurposed as a digital picture frame.

## Configuration

The configuraion file must be created in the same directory where the randompic executable is run from.

Example of the config file from the repo and the corresponding values is seen below:

```bash
{
    "excludedExtensions": [".mp4", ".mov", ".heic"],            # a list of strings containing the file extensions to exclude from display
    "excludedDirectories": ["2022-11-07"],                      # a list of strings present in teh directories to exclude from being loaded
    "imageDirectory": "/mnt/photos",                            # the absolute path to the directory to load the images from, in string format
    "displaySeconds": 15                                        # an integer value in seconds which is the amount of time to display the image before moving to the next one
}
```