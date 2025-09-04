package captcha

import (
	"context"
	"encoding/json"
	"math/rand"
	"os"
	"slices"
	"sync"
	"time"
)

type CaptchaTask struct {
	Id          uint32 `json:"id"`
	Description string `json:"description,omitempty"`
	Complexity  int    `json:"complexity,omitempty"`
	Images      []string
	Solution    []int
	ts          time.Time
}

type CaptchaSolution struct {
	Id       uint32 `json:"id"`
	Solution []int  `json:"solution,omitempty"`
}

type templatesConfiguration struct {
	Templates []CaptchaTask `json:"templates"`
}

type ClassificationCaptchaManager struct {
	ctx                        context.Context
	temlates                   []CaptchaTask
	tasks                      map[uint32]*CaptchaTask
	complexity                 int
	templatesConfigurationPath string
	mu                         sync.Mutex
}

func (c *ClassificationCaptchaManager) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, task := range c.tasks {
		if time.Since(task.ts) > time.Hour { // todo: debug
			delete(c.tasks, id)
		}
	}
}

func (c *ClassificationCaptchaManager) Serve() (err error) {
	// Load templates configuration
	var templates templatesConfiguration
	content, err := os.ReadFile(c.templatesConfigurationPath)
	if err != nil {
		return
	}
	err = json.Unmarshal(content, &templates)
	if err != nil {
		return
	}
	c.mu.Lock()
	c.temlates = templates.Templates
	c.mu.Unlock()

	// Cleanup procedure
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-c.ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			c.cleanup()
		}
	}
}

func (c *ClassificationCaptchaManager) GetTask() *CaptchaTask {
	taskTemplateIndex := rand.Intn(len(c.temlates))
	template := c.temlates[taskTemplateIndex]

	task := CaptchaTask{
		Description: template.Description,
		Images:      make([]string, c.complexity),
		Complexity:  c.complexity,
		ts:          time.Now(),
	}

	shuffledIndex := rand.Perm(c.complexity)
	task.Solution = slices.Clone(shuffledIndex[:c.complexity/2])
	slices.Sort(task.Solution)
	for i := 0; i < c.complexity; i++ {
		index := shuffledIndex[i]
		if i < c.complexity/2 {
			task.Images[index] = template.Images[index]

		} else {
			for {
				templateIndex := rand.Intn(len(c.temlates))
				if templateIndex != taskTemplateIndex {
					task.Images[index] = c.temlates[templateIndex].Images[index]
					break
				}
			}
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	for {
		task.Id = rand.Uint32()
		if _, exists := c.tasks[task.Id]; !exists {
			c.tasks[task.Id] = &task
			break
		}
	}
	return &task
}

func (c *ClassificationCaptchaManager) ResolveFile(taskId uint32, imageId int) (file string, exists bool) {
	task, exists := c.tasks[taskId]
	if !exists {
		return
	}
	if imageId < 0 || imageId >= len(task.Images) {
		return
	}
	return task.Images[imageId], true
}

func (c *ClassificationCaptchaManager) Verify(solution *CaptchaSolution) bool {
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

func NewClassificationCaptchaManager(ctx context.Context, complexity int, config string) *ClassificationCaptchaManager {
	manager := ClassificationCaptchaManager{
		ctx:                        ctx,
		complexity:                 complexity,
		templatesConfigurationPath: config,
		tasks:                      make(map[uint32]*CaptchaTask),
	}

	return &manager
}
