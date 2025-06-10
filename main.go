package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Version is set during build time
var Version = "dev"

type AppConfig struct {
	MediaDir     string
	S3Bucket     string
	S3Region     string
	SyncInterval time.Duration
	Port         string
}

type MediaFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	URL  string `json:"url"`
}

type Server struct {
	config    AppConfig
	s3Client  *s3.Client
	mediaList []MediaFile
}

func main() {
	var (
		showVersion = flag.Bool("version", false, "Show version information")
		showHelp    = flag.Bool("help", false, "Show help information")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("Digital Signage %s\n", Version)
		return
	}

	if *showHelp {
		fmt.Printf("Digital Signage %s\n\n", Version)
		fmt.Println("A lightweight digital signage application")
		fmt.Println("\nUsage:")
		fmt.Println("  digital-signage [options]")
		fmt.Println("\nOptions:")
		fmt.Println("  --version    Show version information")
		fmt.Println("  --help       Show this help message")
		fmt.Println("\nEnvironment Variables:")
		fmt.Println("  MEDIA_DIR              Directory containing video files (default: ./media)")
		fmt.Println("  PORT                   HTTP server port (default: 8080)")
		fmt.Println("  S3_BUCKET              S3 bucket name for sync (optional)")
		fmt.Println("  S3_REGION              AWS region (default: us-east-1)")
		fmt.Println("  SYNC_INTERVAL_MINUTES  S3 sync interval in minutes (default: 15)")
		fmt.Println("  AWS_ACCESS_KEY_ID      AWS access key (optional)")
		fmt.Println("  AWS_SECRET_ACCESS_KEY  AWS secret key (optional)")
		return
	}

	appconfig := AppConfig{
		MediaDir:     getEnv("MEDIA_DIR", "./media"),
		S3Bucket:     getEnv("S3_BUCKET", ""),
		S3Region:     getEnv("S3_REGION", "sa-east-1"),
		SyncInterval: time.Duration(getEnvInt("SYNC_INTERVAL_MINUTES", 15)) * time.Minute,
		Port:         getEnv("PORT", "8080"),
	}

	// Create media directory if it doesn't exist
	if err := os.MkdirAll(appconfig.MediaDir, 0755); err != nil {
		log.Fatalf("Failed to create media directory: %v", err)
	}

	server := &Server{config: appconfig}

	// Initialize S3 client if bucket is configured
	if appconfig.S3Bucket != "" {
		ctx := context.Background()
		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(appconfig.S3Region))
		if err != nil {
			log.Printf("Failed to load S3 config: %v", err)
		} else {
			server.s3Client = s3.NewFromConfig(cfg)
			log.Println("S3 sync enabled")
		}
	}

	// Initial media scan
	server.scanMedia()

	// Start background sync if S3 is configured
	if server.s3Client != nil {
		go server.syncLoop()
	}

	// Setup HTTP routes
	http.HandleFunc("/", server.handleIndex)
	http.HandleFunc("/api/media", server.handleMediaAPI)
	http.Handle("/media/", http.StripPrefix("/media/", http.FileServer(http.Dir(appconfig.MediaDir))))

	log.Printf("Digital Signage %s starting on port %s", Version, appconfig.Port)
	log.Printf("Media directory: %s", appconfig.MediaDir)
	if appconfig.S3Bucket != "" {
		log.Printf("S3 sync: %s (every %v)", appconfig.S3Bucket, appconfig.SyncInterval)
	}

	if err := http.ListenAndServe(":"+appconfig.Port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Digital Signage</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            background: #000;
            font-family: Arial, sans-serif;
            overflow: hidden;
            cursor: none;
        }
        
        #video-container {
		    width: 100vw;
		    height: 100vh;
		    display: flex;
		    align-items: center;
		    justify-content: center;
		    overflow: hidden;
        }

        video {
            width: auto;
            height: auto;
            max-height: 100%;
            max-width: 100%;
            object-fit: contain;
        }

        #loading {
            position: absolute;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            color: white;
            font-size: 24px;
            text-align: center;
        }
        
        #status {
            position: absolute;
            bottom: 20px;
            right: 20px;
            color: rgba(255, 255, 255, 0.7);
            font-size: 12px;
            background: rgba(0, 0, 0, 0.5);
            padding: 5px 10px;
            border-radius: 3px;
        }
        
        .hidden {
            display: none;
        }
    </style>
</head>
<body>
    <div id="loading">Loading media...</div>
    <div id="video-container" class="hidden">
        <video id="video" muted autoplay></video>
    </div>
    <div id="status">Initializing...</div>

    <script>
        class DigitalSignage {
            constructor() {
                this.mediaList = [];
                this.currentIndex = 0;
                this.video = document.getElementById('video');
                this.loading = document.getElementById('loading');
                this.container = document.getElementById('video-container');
                this.status = document.getElementById('status');
                
                this.init();
            }
            
            async init() {
                try {
                    await this.loadMediaList();
                    this.setupVideo();
                    this.hideLoading();
                    this.startPlayback();
                    this.startMediaRefresh();
                } catch (error) {
                    console.error('Initialization failed:', error);
                    this.showError('Failed to load media');
                }
            }
            
            async loadMediaList() {
                const response = await fetch('/api/media');
                const data = await response.json();
                this.mediaList = data.media || [];
                this.updateStatus(` + "`" + `${this.mediaList.length} media files loaded` + "`" + `);
            }
            
            setupVideo() {
                this.video.addEventListener('ended', () => {
                    this.playNext();
                });
                
                this.video.addEventListener('error', (e) => {
                    console.error('Video error:', e);
                    this.playNext();
                });
                
                this.video.addEventListener('loadstart', () => {
                    this.updateStatus('Loading video...');
                });
                
                this.video.addEventListener('canplay', () => {
                    this.updateStatus(` + "`" + `Playing: ${this.getCurrentMedia().name}` + "`" + `);
                });
            }
            
            hideLoading() {
                this.loading.classList.add('hidden');
                this.container.classList.remove('hidden');
            }
            
            showError(message) {
                this.loading.textContent = message;
                this.updateStatus(message);
            }
            
            getCurrentMedia() {
                return this.mediaList[this.currentIndex] || null;
            }
            
            async startPlayback() {
                if (this.mediaList.length === 0) {
                    this.showError('No media files found');
                    return;
                }
                
                this.playCurrentMedia();
            }
            
            async playCurrentMedia() {
                const media = this.getCurrentMedia();
                if (!media) return;
                
                this.video.src = media.url;
                try {
                    await this.video.play();
                } catch (error) {
                    console.error('Play failed:', error);
                    setTimeout(() => this.playNext(), 1000);
                }
            }
            
            playNext() {
                if (this.mediaList.length === 0) return;
                
                this.currentIndex = (this.currentIndex + 1) % this.mediaList.length;
                this.playCurrentMedia();
            }
            
            updateStatus(message) {
                this.status.textContent = message;
            }
            
            startMediaRefresh() {
                // Refresh media list every 5 minutes
                setInterval(async () => {
                    try {
                        const oldCount = this.mediaList.length;
                        await this.loadMediaList();
                        
                        if (this.mediaList.length !== oldCount) {
                            console.log('Media list updated');
                            // Reset to beginning if current index is out of bounds
                            if (this.currentIndex >= this.mediaList.length) {
                                this.currentIndex = 0;
                                this.playCurrentMedia();
                            }
                        }
                    } catch (error) {
                        console.error('Failed to refresh media list:', error);
                    }
                }, 5 * 60 * 1000);
            }
        }
        
        // Start the application
        document.addEventListener('DOMContentLoaded', () => {
            new DigitalSignage();
        });
        
        // Prevent context menu and other interactions
        document.addEventListener('contextmenu', e => e.preventDefault());
        document.addEventListener('keydown', e => {
            if (e.key === 'F5' || (e.ctrlKey && e.key === 'r')) {
                e.preventDefault();
            }
        });
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, tmpl)
}

func (s *Server) handleMediaAPI(w http.ResponseWriter, r *http.Request) {
	s.scanMedia()

	response := map[string]interface{}{
		"media": s.mediaList,
		"count": len(s.mediaList),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) scanMedia() {
	var mediaFiles []MediaFile
	supportedExts := map[string]bool{
		".mp4": true, ".avi": true, ".mov": true, ".mkv": true,
		".webm": true, ".m4v": true, ".3gp": true,
	}

	err := filepath.Walk(s.config.MediaDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if supportedExts[ext] {
				relPath, _ := filepath.Rel(s.config.MediaDir, path)
				mediaFile := MediaFile{
					Name: info.Name(),
					Path: path,
					URL:  "/media/" + filepath.ToSlash(relPath),
				}
				mediaFiles = append(mediaFiles, mediaFile)
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("Error scanning media directory: %v", err)
	}

	// Sort by name for consistent playback order
	sort.Slice(mediaFiles, func(i, j int) bool {
		return mediaFiles[i].Name < mediaFiles[j].Name
	})

	s.mediaList = mediaFiles
	log.Printf("Found %d media files", len(mediaFiles))
}

func (s *Server) syncLoop() {
	log.Println("Starting S3 sync loop")

	// Initial sync
	s.syncFromS3()

	// Periodic sync
	ticker := time.NewTicker(s.config.SyncInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.syncFromS3()
	}
}

func (s *Server) syncFromS3() {
	if s.s3Client == nil {
		return
	}

	log.Println("Starting S3 sync...")
	ctx := context.Background()

	// List objects in S3 bucket
	resp, err := s.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.config.S3Bucket),
	})
	if err != nil {
		log.Printf("Failed to list S3 objects: %v", err)
		return
	}

	localFilesToRemove := make([]string, len(s.mediaList))
	for i := range len(s.mediaList) {
		localFilesToRemove[i] = s.mediaList[i].Path
	}
	syncCount := 0
	for _, obj := range resp.Contents {
		if obj.Key == nil {
			continue
		}

		fileName := *obj.Key
		localPath := filepath.Join(s.config.MediaDir, fileName)

		// Check if file exists
		if _, err := os.Stat(localPath); err == nil {
			// Delete from known localfiles
			index := slices.Index(localFilesToRemove, localPath)
			if index != -1 {
				localFilesToRemove = slices.Delete(localFilesToRemove, index, index+1)
			}
			continue
		}
		// // Check if file exists and has same size
		// if info, err := os.Stat(localPath); err == nil {
		// 	if info.Size() == obj.Size {
		// 		continue // File already exists with same size
		// 	}
		// }

		// Download file
		if err := s.downloadFromS3(ctx, fileName, localPath); err != nil {
			log.Printf("Failed to download %s: %v", fileName, err)
			continue
		}

		syncCount++
		log.Printf("Downloaded: %s", fileName)
	}

	if len(localFilesToRemove) > 0 {
		log.Printf("%d files were deleted from S3 and need to be deleted from local storage", len(localFilesToRemove))
		for _, localF := range localFilesToRemove {
			os.Remove(localF)
		}
	}

	if syncCount > 0 {
		log.Printf("S3 sync completed: %d files updated", syncCount)
		s.scanMedia() // Refresh media list
	} else {
		log.Println("S3 sync completed: no updates needed")
	}
}

func (s *Server) downloadFromS3(ctx context.Context, key, localPath string) error {
	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}

	// Download from S3
	resp, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.config.S3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create local file
	file, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Copy data
	_, err = io.Copy(file, resp.Body)
	return err
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
