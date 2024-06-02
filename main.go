package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/corona10/goimagehash"
	"github.com/gin-gonic/gin"
	"github.com/kyroy/kdtree"
	"github.com/kyroy/kdtree/points"
	"gopkg.in/go-dedup/simhash.v2"
)

// ImageHash wraps the hash and filepath
type ImageHash struct {
	FilePath string
	Hash     uint64
}

type HashPoint struct {
	points.Point
	filePath string
	Hash     uint64
}

// Helper function to create a new HashPoint
func NewHashPoint(path string, hash uint64) *HashPoint {
	// Convert the hash to a single-dimensional float64 slice for the KD-tree.
	coords := []float64{float64(hash)}
	return &HashPoint{
		Point:    *points.NewPoint(coords, nil),
		Hash:     hash,
		filePath: path,
	}
}

// HashImage reads an image file and computes its Simhash.
func HashImage(filePath string) (uint64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return 0, err
	}

	hash, err := goimagehash.PerceptionHash(img)
	if err != nil {
		return 0, err
	}

	// Get the 64-bit hash value
	hashValue := hash.GetHash()

	// Convert the hash value to a slice of bytes
	hashBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(hashBytes, hashValue)

	// Create a feature set from the hash bytes
	features := [][]byte{hashBytes} // Simhash expects a slice of byte slices

	// Use the SimhashBytes function to compute the Simhash
	simhasher := simhash.NewSimhash()
	simhashValue := simhasher.SimhashBytes(features)
	fmt.Println(simhashValue, hashValue)
	return simhashValue, nil

}

// WriteHashesToFile writes the image file paths and their corresponding hashes to a file.
func WriteHashesToFile(hashes []ImageHash, filePath string) error {
	// Open the file with flags to append and create if it doesn't exist
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, hash := range hashes {
		// Each line contains the file path and the hash in hexadecimal
		line := fmt.Sprintf("%s %x\n", hash.FilePath, hash.Hash)
		_, err := writer.WriteString(line)
		if err != nil {
			return err
		}
	}

	return writer.Flush()
}

func LoadHashesFromFile(filePath string) ([]*HashPoint, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var points []*HashPoint
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var path string
		var hash uint64
		fmt.Sscanf(scanner.Text(), "%s %x", &path, &hash)
		points = append(points, NewHashPoint(path, hash))
	}
	return points, scanner.Err()
}

var tree *kdtree.KDTree

func main() {
	hitit()
	routes := gin.New()
	handlers := routes.Group("api")
	{
		// handlers.GET("/cat", cat) //checked->
		// handlers.GET("/verifyWebhook", verifyWebhook)
		// handlers.GET("/handleWebhook", handleWebhook)
		handlers.GET("/cat/:hash", cat)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	err := routes.Run(":" + port)
	if err != nil {
		fmt.Println(err.Error())
	}
}
func hitit() {
	// imagesDir := "/Users/shivamsouravjha/cat/images"
	outputFile := "image_hashes.txt"
	// var count int

	// var imageHashes []ImageHash
	// err := filepath.Walk(imagesDir, func(path string, info os.FileInfo, err error) error {
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if !info.IsDir() && !strings.HasSuffix(info.Name(), ".jpg.cat") {
	// 		hash, err := HashImage(path)
	// 		if err != nil {
	// 			log.Printf("Failed to hash image %s: %v", path, err)
	// 			return nil // Continue processing other files
	// 		}
	// 		imageHashes = append(imageHashes, ImageHash{FilePath: path, Hash: hash})
	// 		count++

	// 		// Write to file every 10,000 images
	// 		if count >= 10000 {
	// 			if err = WriteHashesToFile(imageHashes, outputFile); err != nil {
	// 				log.Printf("Failed to write hashes to file: %v", err)
	// 			}
	// 			imageHashes = []ImageHash{} // Reset slice
	// 			count = 0
	// 		}
	// 	}
	// 	return nil
	// })

	// if err != nil {
	// 	log.Fatalf("Failed to process images: %v", err)
	// }

	// //Write any remaining hashes that didn't make up a full chunk
	// if len(imageHashes) > 0 {
	// 	if err = WriteHashesToFile(imageHashes, outputFile); err != nil {
	// 		log.Fatalf("Failed to write remaining hashes to file: %v", err)
	// 	}
	// }
	// Load hashes into KD-tree

	hashPoints, err := LoadHashesFromFile(outputFile)
	if err != nil {
		log.Fatalf("Failed to load hashes from file: %v", err)
	}
	tree = kdtree.New([]kdtree.Point{})
	for count, point := range hashPoints {
		fmt.Println("here", count)
		tree.Insert(point)
	}
}

func cat(c *gin.Context) {
	anjaliHash, err := strconv.ParseUint(c.Params.ByName("hash"), 16, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid hash value"})
		return
	}

	// var anjaliHash ImageHash
	filePath := filepath.Join(".", "downloaded_image.jpg")
	absFilePath, _ := filepath.Abs(filePath)
	anjali := absFilePath

	// // var imageHashes []ImageHash
	// err := filepath.Walk(anjali, func(path string, info os.FileInfo, err error) error {
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if !info.IsDir() && !strings.HasSuffix(info.Name(), ".jpg.cat") {
	// 		hash, err := HashImage(path)
	// 		if err != nil {
	// 			log.Printf("Failed to hash image %s: %v", path, err)
	// 			return nil // Continue processing other files
	// 		}
	// 		anjaliHash = ImageHash{
	// 			FilePath: path,
	// 			Hash:     hash,
	// 		}
	// 	}
	// 	return nil
	// })
	// fmt.Println(err)
	var points = NewHashPoint(anjali, anjaliHash)

	fmt.Println("herehereherehereherehereherehereherehereherehereherehereherehereherehereherehere")
	// Find the nearest neighbor
	nearest := tree.KNN(points, 1) // Adjust the query as needed
	fmt.Printf("Nearest hash to %v is %v\n", nearest[0].(*HashPoint).filePath)

	fmt.Println("KD-tree search completed.")

	fmt.Println("Hashes written to file successfully.")
	// err = os.Remove(anjali)
	// if err != nil {
	// 	log.Printf("Failed to delete image %s: %v", anjali, err)
	// }

	c.JSON(200, nearest[0].(*HashPoint).filePath)
}

func downloadImage(mediaUrl, phoneNumber string) error {
	accessToken := os.Getenv("WHATSAPP_ACCESS_TOKEN")
	client := resty.New()

	var downloadResp *resty.Response
	var err error

	// Retry mechanism (consider increasing retries based on needs)
	for retries := 0; retries < 3; retries++ {
		downloadResp, err = client.R().
			SetHeader("Authorization", "Bearer "+accessToken).
			Get(mediaUrl)

		if err == nil && downloadResp.StatusCode() == http.StatusOK {
			break
		}

		fmt.Printf("Retrying download... attempt %d\n", retries+1)
		time.Sleep(2 * time.Second) // Wait for 2 seconds before retrying
	}

	if err != nil {
		return fmt.Errorf("error downloading image: %v", err)
	}

	if downloadResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("failed to download image, status code: %d", downloadResp.StatusCode())
	}

	filePath := filepath.Join(".", "downloaded_image.jpg")
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("error getting absolute file path: %v", err)
	}

	file, err := os.Create(absFilePath)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	// Read the entire response body (consider chunking for large files)
	bodyBytes, err := ioutil.ReadAll(downloadResp.RawBody())
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	// Write the body to file
	_, err = file.Write(bodyBytes)
	if err != nil {
		return fmt.Errorf("error saving image: %v", err)
	}

	fmt.Printf("Downloaded image for phone number: %s to path: %s\n", phoneNumber, absFilePath)
	return nil
}

func getMediaUrl(mediaId string) (string, error) {
	accessToken := os.Getenv("WHATSAPP_ACCESS_TOKEN")
	url := fmt.Sprintf("https://graph.facebook.com/v16.0/%s", mediaId)

	client := resty.New()
	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+accessToken).
		Get(url)

	if err != nil {
		return "", fmt.Errorf("error getting media URL: %v", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("failed to get media URL, status code: %d", resp.StatusCode())
	}

	var mediaData map[string]interface{}
	err = json.Unmarshal(resp.Body(), &mediaData)
	if err != nil {
		return "", fmt.Errorf("error parsing media URL JSON: %v", err)
	}

	mediaUrl, urlExists := mediaData["url"].(string)
	if !urlExists {
		return "", fmt.Errorf("no URL found in media data")
	}

	return mediaUrl, nil
}

// func sendMessage(phoneNumber, message string) {
// 	accessToken := os.Getenv("WHATSAPP_ACCESS_TOKEN")
// 	phoneNumberId := os.Getenv("WHATSAPP_PHONE_NUMBER_ID")
// 	client := resty.New()
// 	_, err := client.R().
// 		SetAuthToken(accessToken).
// 		SetBody(map[string]interface{}{
// 			"messaging_product": "whatsapp",
// 			"to":                phoneNumber,
// 			"text":              map[string]string{"body": message},
// 		}).
// 		Post(fmt.Sprintf("https://graph.facebook.com/v13.0/%s/messages", phoneNumberId))

// 	if err != nil {
// 		fmt.Printf("Error sending message: %v\n", err)
// 	}
// }
// func handleWebhook(c *gin.Context) {
// 	// err := godotenv.Load()
// 	// if err != nil {
// 	// 	fmt.Println("Error loading .env file")
// 	// 	c.String(http.StatusInternalServerError, "Internal Server Error")
// 	// 	return
// 	// }

// 	mode := c.Query("hub.mode")
// 	token := c.Query("hub.verify_token")
// 	challenge := c.Query("hub.challenge")
// 	verifyToken := os.Getenv("WHATSAPP_VERIFY_TOKEN")
// 	if mode == "subscribe" && token == verifyToken {
// 		c.String(http.StatusOK, challenge)
// 		return
// 	}

// 	if c.Request.Method == http.MethodPost {
// 		var body map[string]interface{}
// 		if err := c.BindJSON(&body); err != nil {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 			return
// 		}

// 		fmt.Printf("Received Webhook: %v\n", body)

// 		entry, entryExists := body["entry"].([]interface{})
// 		if !entryExists || len(entry) == 0 {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "No entry found"})
// 			return
// 		}

// 		changes, changesExists := entry[0].(map[string]interface{})["changes"].([]interface{})
// 		if !changesExists || len(changes) == 0 {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "No changes found"})
// 			return
// 		}

// 		value, valueExists := changes[0].(map[string]interface{})["value"].(map[string]interface{})
// 		if !valueExists {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "No value found"})
// 			return
// 		}

// 		messages, messagesExists := value["messages"].([]interface{})
// 		if !messagesExists || len(messages) == 0 {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "No messages found"})
// 			return
// 		}

// 		message := messages[0].(map[string]interface{})
// 		messageType, messageTypeExists := message["type"].(string)
// 		if !messageTypeExists || messageType != "image" {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "Message is not an image"})
// 			return
// 		}

// 		image, imageExists := message["image"].(map[string]interface{})
// 		if !imageExists {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "No image found in message"})
// 			return
// 		}

// 		imageId, idExists := image["id"].(string)
// 		if !idExists {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "No image ID found"})
// 			return
// 		}

// 		phoneNumber, phoneExists := message["from"].(string)
// 		if !phoneExists {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "No phone number found"})
// 			return
// 		}

// 		mediaUrl, err := getMediaUrl(imageId)
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 			return
// 		}

// 		err = downloadImage(mediaUrl, phoneNumber)
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 			return
// 		}
// 		// nearest := cat(imageId)
// 		// sendMessage(phoneNumber, nearest)

// 		c.Status(http.StatusOK)
// 	}
// }
