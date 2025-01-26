# randompic

A basic go app that loads images from a directory and displays the image on a web page.

The app takes a local directory path as input, cleans up the file path (strips the path and starts a file server to serve images) and selected an image at random to display.

Every 10 seconds a new random image is selected to be displayed and the page automatically updated.

Photos are served from a directory on server/computer where the app runs, or images could be mounted from a network share on the computer where the app is run from.

This page can be opened/displayed on an old tablet/device so it can be repurposed as a digital picture frame.

**NOTE:** The photo directory is hard coded and would need to be manually changed in the code to your desired location.