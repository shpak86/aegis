package captcha

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"math/rand"
	"os"
	"slices"
	"sync"
	"time"
)

type ChallengeTemplate struct {
	Description string   `json:"description,omitempty"`
	Images      []string `json:"images"`
}

type Challenge struct {
	Description  string `json:"description,omitempty"`
	Id           uint32 `json:"id"`
	Complexity   int    `json:"complexity,omitempty"`
	Solution     []int
	Base64Images [][]byte
	ts           time.Time
}

type Solution struct {
	Id       uint32 `json:"id"`
	Solution []int  `json:"solution,omitempty"`
}

type Configuration struct {
	Templates []ChallengeTemplate `json:"templates"`
}

type CaptchaManager struct {
	ctx          context.Context
	templates    []ChallengeTemplate
	tasks        map[uint32]*Challenge
	complexity   int
	base64Images map[string]string
	mu           sync.Mutex
}

func (c *CaptchaManager) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, task := range c.tasks {
		if time.Since(task.ts) > time.Minute {
			delete(c.tasks, id)
		}
	}
}

func (c *CaptchaManager) Serve() (err error) {
	// Cleanup and update procedures
	cleanuupTicker := time.NewTicker(time.Second)
	loadTicker := time.NewTicker(time.Minute)
	for {
		select {
		case <-c.ctx.Done():
			cleanuupTicker.Stop()
			return
		case <-cleanuupTicker.C:
			c.cleanup()
		case <-loadTicker.C:
			c.load()
		}
	}
}

func (c *CaptchaManager) GetChallenge() *Challenge {
	challengeIdx := rand.Intn(len(c.templates))
	challengeTemplate := c.templates[challengeIdx]
	challenge := Challenge{
		Description:  challengeTemplate.Description,
		Base64Images: make([][]byte, c.complexity),
		Complexity:   c.complexity,
		ts:           time.Now(),
	}
	shuffledIndex := rand.Perm(c.complexity)
	challenge.Solution = slices.Clone(shuffledIndex[:c.complexity/2])
	slices.Sort(challenge.Solution)
	for i := 0; i < c.complexity; i++ {
		index := shuffledIndex[i]
		if i < c.complexity/2 {
			imageKey := fmt.Sprintf("%d:%d", challengeIdx, index)
			challenge.Base64Images[index] = []byte(c.base64Images[imageKey])
		} else {
			for {
				templateIndex := rand.Intn(len(c.templates))
				if templateIndex != challengeIdx {
					imageKey := fmt.Sprintf("%d:%d", templateIndex, index)
					challenge.Base64Images[index] = []byte(c.base64Images[imageKey])
					break
				}
			}
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	for {
		challenge.Id = rand.Uint32()
		if _, exists := c.tasks[challenge.Id]; !exists {
			c.tasks[challenge.Id] = &challenge
			break
		}
	}
	return &challenge
}

func (c *CaptchaManager) Verify(solution *Solution) bool {
	task, exists := c.tasks[solution.Id]
	if !exists {
		return false
	}
	isValid := slices.Equal(task.Solution, solution.Solution)
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.tasks, solution.Id)
	return isValid
}

func (c *CaptchaManager) load() (err error) {
	// Load configuration from file
	content, err := os.ReadFile("/etc/aegis/captcha.json")
	if err != nil {
		return
	}
	var configuration Configuration
	err = json.Unmarshal(content, &configuration)
	if err != nil {
		return
	}
	// Load images
	buf := bytes.Buffer{}
	images := map[string]string{}
	for templateId, template := range configuration.Templates {
		for imageId, imageFile := range template.Images {
			var f *os.File
			f, err = os.Open(imageFile)
			if err != nil {
				return
			}
			defer f.Close()

			img, _ := jpeg.Decode(f)
			noisedImage := addUniformNoise(img, 20)
			buf.Reset()
			err = jpeg.Encode(&buf, noisedImage, &jpeg.Options{Quality: 50})
			if err != nil {
				return
			}
			images[fmt.Sprintf("%d:%d", templateId, imageId)] = base64.StdEncoding.EncodeToString(buf.Bytes())
		}
	}
	// Update current images
	c.mu.Lock()
	defer c.mu.Unlock()
	c.templates = configuration.Templates
	c.base64Images = images
	return
}

func NewClassificationCaptchaManager(ctx context.Context, complexity int) *CaptchaManager {
	manager := CaptchaManager{
		ctx:          ctx,
		complexity:   complexity,
		tasks:        make(map[uint32]*Challenge),
		base64Images: make(map[string]string),
	}
	manager.load()
	go manager.Serve()
	return &manager
}
