package ssg

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cliossg/clio/internal/feat/profile"
	"github.com/cliossg/clio/pkg/cl/logger"
	"github.com/google/uuid"
)

type SeederProfileService interface {
	CreateProfile(ctx context.Context, slug, name, surname, bio, socialLinks, photoPath, createdBy string) (*profile.Profile, error)
}

type Seeder struct {
	service        Service
	profileService SeederProfileService
	log            logger.Logger
}

func NewSeeder(service Service, profileService SeederProfileService, log logger.Logger) *Seeder {
	return &Seeder{
		service:        service,
		profileService: profileService,
		log:            log,
	}
}

func (s *Seeder) Start(ctx context.Context) error {
	sites, err := s.service.ListSites(ctx)
	if err != nil {
		return fmt.Errorf("cannot list sites: %w", err)
	}

	if len(sites) > 0 {
		s.log.Info("Sites already exist, skipping SSG seeding")
		return nil
	}

	site, err := s.seedDemoSite(ctx)
	if err != nil {
		return fmt.Errorf("cannot seed demo site: %w", err)
	}

	if err := s.seedDemoContent(ctx, site); err != nil {
		return fmt.Errorf("cannot seed demo content: %w", err)
	}

	s.log.Infof("Seeded demo site: %s", site.Name)
	return nil
}

func (s *Seeder) seedDemoSite(ctx context.Context) (*Site, error) {
	site := NewSite("Demo", "demo", "blog")
	if err := s.service.CreateSite(ctx, site); err != nil {
		return nil, err
	}

	// Create root section
	root := NewSection(site.ID, "/ (root)", "Root section for top-level content", "")
	if err := s.service.CreateSection(ctx, root); err != nil {
		return nil, err
	}

	// Create content sections
	sections := []struct {
		name, description, path string
	}{
		{"Coding", "Programming tutorials and tech articles", "coding"},
		{"Essays", "Personal essays and reflections", "essays"},
		{"Food", "Recipes and culinary adventures", "food"},
	}

	for _, sec := range sections {
		section := NewSection(site.ID, sec.name, sec.description, sec.path)
		if err := s.service.CreateSection(ctx, section); err != nil {
			return nil, err
		}
	}

	// Seed default params
	if err := s.seedDefaultParams(ctx, site.ID); err != nil {
		return nil, err
	}

	// Seed contributors
	if err := s.seedContributors(ctx, site.ID); err != nil {
		return nil, err
	}

	return site, nil
}

func (s *Seeder) seedContributors(ctx context.Context, siteID uuid.UUID) error {
	contributors := []struct {
		handle, name, surname, bio, role string
		socialLinks                      []SocialLink
	}{
		{
			handle:  "johndoe",
			name:    "John",
			surname: "Doe",
			bio:     "Senior software engineer and Go enthusiast. Building tools for developers and writing about backend architecture.",
			role:    "editor",
			socialLinks: []SocialLink{
				{Platform: "GitHub", URL: "https://github.com/johndoe"},
				{Platform: "X", URL: "https://x.com/johndoe"},
				{Platform: "LinkedIn", URL: "https://linkedin.com/in/johndoe"},
			},
		},
		{
			handle:  "janesmith",
			name:    "Jane",
			surname: "Smith",
			bio:     "Technical writer and documentation specialist. Passionate about making complex topics accessible to everyone.",
			role:    "author",
			socialLinks: []SocialLink{
				{Platform: "GitHub", URL: "https://github.com/janesmith"},
				{Platform: "X", URL: "https://x.com/janesmith"},
				{Platform: "Website", URL: "https://janesmith.dev"},
			},
		},
	}

	for _, c := range contributors {
		socialLinksJSON, err := json.Marshal(c.socialLinks)
		if err != nil {
			return fmt.Errorf("cannot marshal social links for %s: %w", c.handle, err)
		}

		p, err := s.profileService.CreateProfile(ctx, c.handle, c.name, c.surname, c.bio, string(socialLinksJSON), "", "")
		if err != nil {
			return fmt.Errorf("cannot create profile for %s: %w", c.handle, err)
		}

		contributor := NewContributor(siteID, c.handle, c.name, c.surname)
		contributor.ProfileID = &p.ID
		contributor.Bio = c.bio
		contributor.Role = c.role
		contributor.SocialLinks = c.socialLinks
		if err := s.service.CreateContributor(ctx, contributor); err != nil {
			return fmt.Errorf("cannot create contributor %s: %w", c.handle, err)
		}
	}

	return nil
}

func (s *Seeder) seedDefaultParams(ctx context.Context, siteID uuid.UUID) error {
	defaults := []struct {
		name        string
		description string
		value       string
		refKey      string
		category    string
		position    int
		system      bool
	}{
		// Site
		{"Site description", "Site description shown in hero and meta", "A personal blog about coding, essays, and food", "site_description", "site", 1, true},
		{"Hero image", "Hero image filename", "", "hero_image", "site", 2, true},
		{"Site base path", "Base path for GitHub Pages subpath hosting", "/", "ssg.site.base_path", "site", 3, true},
		// Display
		{"Index max items", "Maximum items shown on index pages", "9", "ssg.index.maxitems", "display", 1, true},
		{"Blocks enabled", "Enable related content blocks", "true", "ssg.blocks.enabled", "display", 2, true},
		{"Blocks max items", "Maximum items shown in content blocks", "5", "ssg.blocks.maxitems", "display", 3, true},
		{"Blocks multi-section", "Show related content from other sections", "true", "ssg.blocks.multisection", "display", 4, true},
		{"Blocks background color", "Background color for related content blocks", "#f0f4f8", "ssg.blocks.bgcolor", "display", 5, true},
		// Search
		{"Google Search enabled", "Enable Google site search", "false", "ssg.search.google.enabled", "search", 1, true},
		{"Google Search ID", "Google Custom Search Engine ID", "", "ssg.search.google.id", "search", 2, true},
		// Git
		{"Publish repository URL", "Git repository URL for publishing", "", "ssg.publish.repo.url", "git", 1, true},
		{"Publish branch", "Git branch for publishing", "gh-pages", "ssg.publish.branch", "git", 2, true},
		{"Publish auth token", "Authentication token for publishing", "", "ssg.publish.auth.token", "git", 3, true},
		{"Backup repository URL", "Git repository URL for markdown backup", "", "ssg.backup.repo.url", "git", 4, true},
		{"Backup branch", "Git branch for markdown backup", "main", "ssg.backup.branch", "git", 5, true},
		{"Backup auth token", "Authentication token for backup", "", "ssg.backup.auth.token", "git", 6, true},
		{"Commit user name", "Git user name for commits", "Clio Bot", "ssg.git.commit.user.name", "git", 7, true},
		{"Commit user email", "Git user email for commits", "clio@localhost", "ssg.git.commit.user.email", "git", 8, true},
	}

	for _, d := range defaults {
		param := NewParam(siteID, d.name, d.value)
		param.Description = d.description
		param.RefKey = d.refKey
		param.Category = d.category
		param.Position = d.position
		param.System = d.system
		if err := s.service.CreateParam(ctx, param); err != nil {
			return fmt.Errorf("cannot create param %s: %w", d.name, err)
		}
	}

	return nil
}

func (s *Seeder) seedDemoContent(ctx context.Context, site *Site) error {
	sections, err := s.service.GetSections(ctx, site.ID)
	if err != nil {
		return err
	}

	sectionMap := make(map[string]*Section)
	for _, sec := range sections {
		sectionMap[sec.Path] = sec
	}

	contributors, err := s.service.GetContributors(ctx, site.ID)
	if err != nil {
		return err
	}

	contributorMap := make(map[string]*Contributor)
	for _, c := range contributors {
		contributorMap[c.Handle] = c
	}

	now := time.Now()

	// Home page
	if rootSection := sectionMap[""]; rootSection != nil {
		home := &Content{
			ID:        uuid.New(),
			SiteID:    site.ID,
			SectionID: rootSection.ID,
			ShortID:   uuid.New().String()[:8],
			Kind:      "page",
			Heading:   "Welcome to Clio",
			Body:      "Clio is a static site generator with a built-in admin interface.\n\n## Features\n\n- Markdown content editing\n- Live preview\n- Image management\n- Multiple sections\n- Tags and categories\n\nStart creating content using the admin panel.",
			Summary:   "Welcome page for the demo site",
			Draft:     false,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := s.service.CreateContent(ctx, home); err != nil {
			return err
		}
	}

	type postData struct {
		heading     string
		body        string
		summary     string
		tags        []string
		contributor string
	}

	codingPosts := []postData{
		{
			heading:     "Getting Started with Go",
			contributor: "johndoe",
			body:    "Go is a statically typed, compiled language designed for simplicity and efficiency.\n\n## Why Go?\n\n- Fast compilation\n- Built-in concurrency with goroutines\n- Simple and clean syntax\n- Excellent standard library\n\n## Hello World\n\n```go\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}\n```\n\n## Next Steps\n\nExplore the standard library and build your first web server!",
			summary: "A beginner's guide to the Go programming language",
			tags:    []string{"golang", "tutorial", "beginner"},
		},
		{
			heading: "Understanding Git Branching",
			body:    "Branching is one of Git's most powerful features.\n\n## Creating a Branch\n\n```bash\ngit checkout -b feature/new-feature\n```\n\n## Common Workflows\n\n1. **Feature branches**: Create a branch for each feature\n2. **Main branch**: Keep it stable and deployable\n3. **Merge often**: Avoid long-lived branches\n\n## Best Practices\n\n- Use descriptive branch names\n- Delete merged branches\n- Review before merging",
			summary: "Master Git branching for better collaboration",
			tags:    []string{"git", "devops", "tutorial"},
		},
		{
			heading: "Building REST APIs",
			body:    "REST APIs are the backbone of modern web applications.\n\n## Key Principles\n\n- **Stateless**: Each request contains all needed information\n- **Resource-based**: URLs represent resources\n- **HTTP methods**: GET, POST, PUT, DELETE\n\n## Example Endpoint\n\n```\nGET /api/users/:id\nPOST /api/users\nPUT /api/users/:id\nDELETE /api/users/:id\n```\n\n## Response Codes\n\n- 200: Success\n- 201: Created\n- 404: Not Found\n- 500: Server Error",
			summary: "Learn the fundamentals of REST API design",
			tags:    []string{"api", "backend", "tutorial"},
		},
		{
			heading: "Introduction to Docker",
			body:    "Docker simplifies application deployment through containerization.\n\n## What is a Container?\n\nA container packages your application with all its dependencies into a single unit.\n\n## Basic Commands\n\n```bash\ndocker build -t myapp .\ndocker run -p 8080:8080 myapp\ndocker ps\n```\n\n## Dockerfile\n\n```dockerfile\nFROM golang:1.21\nWORKDIR /app\nCOPY . .\nRUN go build -o main .\nCMD [\"./main\"]\n```\n\n## Benefits\n\n- Consistent environments\n- Easy scaling\n- Fast deployment",
			summary: "Get started with containerization using Docker",
			tags:    []string{"docker", "devops", "beginner"},
		},
	}

	essaysPosts := []postData{
		{
			heading: "The Art of Simplicity",
			body:    "In a world of increasing complexity, simplicity becomes a superpower.\n\n## Less is More\n\nWe often add features, options, and complexity without questioning whether they're necessary. The best solutions are often the simplest ones.\n\n## Practical Simplicity\n\n- Remove what doesn't add value\n- Focus on the essential\n- Embrace constraints\n\n## The Paradox\n\nAchieving simplicity is hard work. It requires deep understanding and the courage to say no.",
			summary: "Reflections on the power of keeping things simple",
			tags:    []string{"philosophy", "productivity", "mindset"},
		},
		{
			heading:     "Learning in Public",
			contributor: "janesmith",
			body:        "Sharing your learning journey can accelerate your growth.\n\n## Why Share?\n\n- Teaching reinforces learning\n- Build connections with others\n- Create a record of progress\n- Help others on similar paths\n\n## How to Start\n\n1. Write about what you learned today\n2. Share your mistakes and lessons\n3. Be authentic and humble\n\n## The Fear\n\nYes, you might be wrong sometimes. That's okay. Growth requires vulnerability.",
			summary:     "Why sharing your learning journey matters",
			tags:        []string{"learning", "growth", "beginner"},
		},
		{
			heading: "Digital Minimalism",
			body:    "Our relationship with technology deserves intentional design.\n\n## The Problem\n\n- Constant notifications\n- Endless scrolling\n- Attention fragmentation\n\n## A Different Approach\n\nBe intentional about technology use:\n\n1. Define your values\n2. Choose tools that support them\n3. Set boundaries\n\n## Practical Steps\n\n- Turn off non-essential notifications\n- Schedule technology-free time\n- Curate your digital environment",
			summary: "Finding balance in our relationship with technology",
			tags:    []string{"technology", "minimalism", "productivity"},
		},
		{
			heading: "The Value of Boredom",
			body:    "We've forgotten how to be bored, and it's costing us.\n\n## The Lost Art\n\nBoredom used to be unavoidable. Now we fill every moment with stimulation.\n\n## What We Lose\n\n- Creativity needs empty space\n- Self-reflection requires silence\n- Ideas need room to breathe\n\n## Reclaiming Boredom\n\n1. Leave your phone behind sometimes\n2. Sit without entertainment\n3. Let your mind wander\n\n## The Gift\n\nBoredom is not the enemy. It's the doorway to deeper thinking.",
			summary: "Why we need more empty space in our minds",
			tags:    []string{"mindfulness", "creativity", "mindset"},
		},
	}

	foodPosts := []postData{
		{
			heading: "Perfect Scrambled Eggs",
			body:    "The secret to great scrambled eggs is patience and low heat.\n\n## Ingredients\n\n- 3 eggs\n- 1 tbsp butter\n- Salt and pepper\n- Fresh chives (optional)\n\n## Method\n\n1. Crack eggs into a cold pan with butter\n2. Place on low heat, stirring constantly\n3. Remove from heat while still slightly wet\n4. Season and serve immediately\n\n## The Secret\n\nNever rush scrambled eggs. Low and slow is the way.",
			summary: "Master the technique for creamy scrambled eggs",
			tags:    []string{"breakfast", "beginner", "technique"},
		},
		{
			heading: "Homemade Pasta Basics",
			body:    "Fresh pasta is simpler than you think.\n\n## Basic Dough\n\n- 100g flour per egg\n- Pinch of salt\n- Knead until smooth\n\n## The Process\n\n1. Make a well in the flour\n2. Add eggs and mix\n3. Knead for 10 minutes\n4. Rest for 30 minutes\n5. Roll and cut\n\n## Tips\n\n- Use semolina flour for better texture\n- Don't skip the resting time\n- Fresh pasta cooks in 2-3 minutes",
			summary: "Learn to make fresh pasta from scratch",
			tags:    []string{"italian", "beginner", "technique"},
		},
		{
			heading:     "The Perfect Cup of Coffee",
			contributor: "johndoe",
			body:        "Great coffee starts with understanding the basics.\n\n## Key Variables\n\n- **Grind size**: Match to your brew method\n- **Water temperature**: 195-205°F (90-96°C)\n- **Ratio**: Start with 1:15 coffee to water\n- **Freshness**: Use beans within 2 weeks of roasting\n\n## Pour Over Method\n\n1. Bloom with twice the coffee weight in water\n2. Wait 30 seconds\n3. Pour in slow circles\n4. Total brew time: 3-4 minutes\n\n## Experiment\n\nAdjust one variable at a time to find your perfect cup.",
			summary:     "Variables and techniques for better coffee at home",
			tags:        []string{"coffee", "brewing", "technique"},
		},
		{
			heading: "Sourdough Bread Basics",
			body:    "Sourdough is both simple and complex. Here's how to start.\n\n## The Starter\n\nA sourdough starter is a living culture of flour and water.\n\n## Feeding Schedule\n\n1. Discard half the starter\n2. Add equal parts flour and water\n3. Wait 12-24 hours\n4. Repeat daily\n\n## Basic Recipe\n\n- 500g flour\n- 350g water\n- 100g active starter\n- 10g salt\n\n## The Process\n\nMix, fold, proof, shape, cold proof overnight, bake at 450°F with steam.\n\n## Patience\n\nGood bread takes time. Embrace the slow process.",
			summary: "Start your sourdough journey with these fundamentals",
			tags:    []string{"baking", "fermentation", "technique"},
		},
	}

	type categoryData struct {
		section string
		posts   []postData
	}

	allPosts := []categoryData{
		{"coding", codingPosts},
		{"essays", essaysPosts},
		{"food", foodPosts},
	}

	postIndex := 0
	for _, category := range allPosts {
		section := sectionMap[category.section]
		if section == nil {
			continue
		}

		for _, p := range category.posts {
			pubTime := now.Add(time.Duration(-postIndex) * 24 * time.Hour)
			post := &Content{
				ID:          uuid.New(),
				SiteID:      site.ID,
				SectionID:   section.ID,
				ShortID:     uuid.New().String()[:8],
				Kind:        "article",
				Heading:     p.heading,
				Body:        p.body,
				Summary:     p.summary,
				Draft:       false,
				PublishedAt: &pubTime,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if p.contributor != "" {
				if c, ok := contributorMap[p.contributor]; ok {
					post.ContributorID = &c.ID
					post.ContributorHandle = c.Handle
				}
			} else {
				post.AuthorUsername = "admin"
			}
			if err := s.service.CreateContent(ctx, post); err != nil {
				return err
			}

			for _, tagName := range p.tags {
				if err := s.service.AddTagToContent(ctx, post.ID, tagName, site.ID); err != nil {
					s.log.Infof("Cannot add tag %s to content: %v", tagName, err)
				}
			}

			postIndex++
		}
	}

	return nil
}

func (s *Seeder) Name() string {
	return "ssg"
}

func (s *Seeder) Depends() []string {
	return []string{"auth"}
}
