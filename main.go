package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-resty/resty/v2"

	"github.com/corona10/goimagehash"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
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
		handlers.GET("/cat", cat) //checked->
		// handlers.GET("/verifyWebhook", verifyWebhook)
		handlers.GET("/handleWebhook", handleWebhook)
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
	var anjaliHash ImageHash
	anjali := "/Users/shivamsouravjha/cat/shivam.jpg"

	// var imageHashes []ImageHash
	err := filepath.Walk(anjali, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !strings.HasSuffix(info.Name(), ".jpg.cat") {
			hash, err := HashImage(path)
			if err != nil {
				log.Printf("Failed to hash image %s: %v", path, err)
				return nil // Continue processing other files
			}
			anjaliHash = ImageHash{
				FilePath: path,
				Hash:     hash,
			}
		}
		return nil
	})
	fmt.Println(err)
	var points = NewHashPoint(anjali, anjaliHash.Hash)

	fmt.Println("herehereherehereherehereherehereherehereherehereherehereherehereherehereherehere")
	// Find the nearest neighbor
	nearest := tree.KNN(points, 1) // Adjust the query as needed
	fmt.Printf("Nearest hash to %v is %v\n", nearest[0].(*HashPoint).filePath)
	c.JSON(200, nearest[0].(*HashPoint).filePath)
	return

	fmt.Println("KD-tree search completed.")

	fmt.Println("Hashes written to file successfully.")
	c.JSON(400, "no")

}
func handleWebhook(c *gin.Context) {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		c.String(http.StatusInternalServerError, "Internal Server Error")
		return
	}

	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")
	verifyToken := os.Getenv("WHATSAPP_VERIFY_TOKEN")
	fmt.Println(mode, token, challenge, verifyToken)
	if mode == "subscribe" && token == verifyToken {
		c.String(http.StatusOK, challenge)
		return
	}

	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entry := body["entry"].([]interface{})[0].(map[string]interface{})
	changes := entry["changes"].([]interface{})[0].(map[string]interface{})
	value := changes["value"].(map[string]interface{})
	messages := value["messages"].([]interface{})[0].(map[string]interface{})

	if messageType, exists := messages["type"].(string); exists && messageType == "image" {
		imageId := messages["image"].(map[string]interface{})["id"].(string)
		phoneNumber := messages["from"].(string)
		downloadImage(imageId, phoneNumber)
		sendMessage(phoneNumber, "Image received and saved.")
	} else {
		phoneNumber := messages["from"].(string)
		sendMessage(phoneNumber, "Please send an image.")
	}

	c.Status(http.StatusOK)
}

func downloadImage(imageId, phoneNumber string) {
	accessToken := os.Getenv("WHATSAPP_ACCESS_TOKEN")

	client := resty.New()
	resp, err := client.R().
		SetAuthToken(accessToken).
		SetOutput(filepath.Join("images", fmt.Sprintf("%s.jpg", imageId))).
		Get(fmt.Sprintf("https://graph.facebook.com/v13.0/%s", imageId))

	if err != nil {
		fmt.Printf("Error downloading image: %v\n", err)
	} else if resp.IsError() {
		fmt.Printf("Error response: %v\n", resp)
	} else {
		fmt.Printf("Image saved for phone number %s\n", phoneNumber)
	}
}

func sendMessage(phoneNumber, message string) {
	accessToken := os.Getenv("WHATSAPP_ACCESS_TOKEN")
	phoneNumberId := os.Getenv("WHATSAPP_PHONE_NUMBER_ID")

	client := resty.New()
	_, err := client.R().
		SetAuthToken(accessToken).
		SetBody(map[string]interface{}{
			"messaging_product": "whatsapp",
			"to":                phoneNumber,
			"text":              map[string]string{"body": message},
		}).
		Post(fmt.Sprintf("https://graph.facebook.com/v13.0/%s/messages", phoneNumberId))

	if err != nil {
		fmt.Printf("Error sending message: %v\n", err)
	}
}
