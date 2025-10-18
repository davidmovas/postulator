package job

import (
	"Postulator/internal/config"
	"Postulator/internal/domain/entities"
	"Postulator/internal/domain/topic"
	"Postulator/internal/infra/database"
	"Postulator/pkg/di"
	"Postulator/pkg/logger"
	"context"
	"os"
	"path/filepath"
	"testing"
)

// --- Fakes for dependencies required by NewService/NewExecutor/NewScheduler ---

type fakeAIClient struct{}

func (f *fakeAIClient) GenerateArticle(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return "generated content", nil
}

// topic service fake
type fakeTopicService struct{}

func (f *fakeTopicService) CreateTopic(ctx context.Context, topic *entities.Topic) (int, error) {
	return 0, nil
}
func (f *fakeTopicService) CreateTopicBatch(ctx context.Context, topics []*entities.Topic) (*topic.BatchCreateResult, error) {
	return &topic.BatchCreateResult{}, nil
}
func (f *fakeTopicService) GetTopic(ctx context.Context, id int64) (*entities.Topic, error) {
	return &entities.Topic{ID: id, Title: "t"}, nil
}
func (f *fakeTopicService) ListTopics(ctx context.Context) ([]*entities.Topic, error) {
	return nil, nil
}
func (f *fakeTopicService) UpdateTopic(ctx context.Context, topic *entities.Topic) error { return nil }
func (f *fakeTopicService) DeleteTopic(ctx context.Context, id int64) error              { return nil }
func (f *fakeTopicService) AssignToSite(ctx context.Context, siteID, topicID, categoryID int64, strategy entities.TopicStrategy) error {
	return nil
}
func (f *fakeTopicService) UnassignFromSite(ctx context.Context, siteID, topicID int64) error {
	return nil
}
func (f *fakeTopicService) GetSiteTopics(ctx context.Context, siteID int64) ([]*entities.SiteTopic, error) {
	return []*entities.SiteTopic{{SiteID: siteID, TopicID: 1, Strategy: entities.StrategyUnique, CategoryID: 1}}, nil
}
func (f *fakeTopicService) GetTopicsBySite(ctx context.Context, siteID int64) ([]*entities.Topic, error) {
	return nil, nil
}
func (f *fakeTopicService) GetAvailableTopic(ctx context.Context, siteID int64, strategy entities.TopicStrategy) (*entities.Topic, error) {
	return &entities.Topic{ID: 1, Title: "topic"}, nil
}
func (f *fakeTopicService) MarkTopicAsUsed(ctx context.Context, siteID, topicID int64) error {
	return nil
}
func (f *fakeTopicService) CountUnusedTopics(ctx context.Context, siteID int64) (int, error) {
	return 42, nil
}

// prompt service fake

type fakePromptService struct{}

func (f *fakePromptService) CreatePrompt(ctx context.Context, prompt *entities.Prompt) error {
	return nil
}
func (f *fakePromptService) GetPrompt(ctx context.Context, id int64) (*entities.Prompt, error) {
	return &entities.Prompt{ID: id, Name: "p", SystemPrompt: "s", UserPrompt: "u"}, nil
}
func (f *fakePromptService) ListPrompts(ctx context.Context) ([]*entities.Prompt, error) {
	return nil, nil
}
func (f *fakePromptService) UpdatePrompt(ctx context.Context, prompt *entities.Prompt) error {
	return nil
}
func (f *fakePromptService) DeletePrompt(ctx context.Context, id int64) error { return nil }
func (f *fakePromptService) RenderPrompt(ctx context.Context, promptID int64, placeholders map[string]string) (string, string, error) {
	return "system", "user", nil
}

// site service fake

type fakeSiteService struct{}

func (f *fakeSiteService) CreateSite(ctx context.Context, site *entities.Site) error { return nil }
func (f *fakeSiteService) GetSite(ctx context.Context, id int64) (*entities.Site, error) {
	return &entities.Site{ID: id, Name: "site", URL: "https://example.com", WPUsername: "u", WPPassword: "p"}, nil
}
func (f *fakeSiteService) GetSiteWithPassword(ctx context.Context, id int64) (*entities.Site, error) {
	return &entities.Site{ID: id, Name: "site", URL: "https://example.com", WPUsername: "u", WPPassword: "p"}, nil
}
func (f *fakeSiteService) ListSites(ctx context.Context) ([]*entities.Site, error)   { return nil, nil }
func (f *fakeSiteService) UpdateSite(ctx context.Context, site *entities.Site) error { return nil }
func (f *fakeSiteService) UpdateSitePassword(ctx context.Context, id int64, password string) error {
	return nil
}
func (f *fakeSiteService) DeleteSite(ctx context.Context, id int64) error         { return nil }
func (f *fakeSiteService) CheckHealth(ctx context.Context, siteID int64) error    { return nil }
func (f *fakeSiteService) SyncCategories(ctx context.Context, siteID int64) error { return nil }
func (f *fakeSiteService) GetSiteCategories(ctx context.Context, siteID int64) ([]*entities.Category, error) {
	return []*entities.Category{{ID: 1, SiteID: siteID, WPCategoryID: 10, Name: "cat"}}, nil
}

type noopExecutor struct{}

func (n *noopExecutor) Execute(ctx context.Context, job *Job) error { return nil }
func (n *noopExecutor) PublishValidatedArticle(ctx context.Context, job *Job, exec *Execution) error {
	return nil
}

// --- Test setup helper ---

func setupJobServiceTest(t *testing.T) (*Service, func()) {
	t.Helper()

	// DB
	db, dbCleanup := database.SetupTestDB(t)

	// Logger
	tempLogDir := filepath.Join(os.TempDir(), "postulator_test_logs", t.Name())
	_ = os.MkdirAll(tempLogDir, 0755)
	tlog, err := logger.NewForTest(&config.Config{LogDir: tempLogDir, AppLogFile: "app.log", ErrLogFile: "err.log", LogLevel: "debug"})
	if err != nil {
		t.Fatalf("logger: %v", err)
	}

	// Seed minimal FK dependencies in DB
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, "INSERT INTO sites(name,url,wp_username,wp_password,status,health_status) VALUES (?,?,?,?,?,?)", "Test Site", "https://example.com", "admin", "password", "active", "unknown"); err != nil {
		t.Fatalf("seed site: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO site_categories(site_id,wp_category_id,name,slug,count) VALUES (?,?,?,?,?)", 1, 10, "Category", "cat", 0); err != nil {
		t.Fatalf("seed category: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO prompts(name,system_prompt,user_prompt,placeholders) VALUES (?,?,?,?)", "P", "S", "U", "[]"); err != nil {
		t.Fatalf("seed prompt: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO ai_providers(name,api_key,provider,model,is_active) VALUES (?,?,?,?,?)", "openai", "sk-test", "openai", "gpt-4o", 1); err != nil {
		t.Fatalf("seed ai provider: %v", err)
	}

	// Minimal DI container only for repos
	c := di.New()
	c.MustRegister(di.Instance[*database.DB](db))
	c.MustRegister(di.Instance[*logger.Logger](tlog))

	jobRepo, err := NewJobRepository(c)
	if err != nil {
		t.Fatalf("job repo: %v", err)
	}
	execRepo, err := NewExecutionRepository(c)
	if err != nil {
		t.Fatalf("exec repo: %v", err)
	}

	// Use real scheduler for CalculateNextRun but no background loop
	sch := &Scheduler{logger: tlog}
	exec := &noopExecutor{}

	svc := &Service{jobRepo: jobRepo, execRepo: execRepo, executor: exec, scheduler: sch, logger: tlog}

	cleanup := func() {
		_ = tlog.Close()
		_ = os.RemoveAll(tempLogDir)
		dbCleanup()
	}
	return svc, cleanup
}

// --- Tests ---

func TestJobService_Create_And_List_Get(t *testing.T) {
	svc, cleanup := setupJobServiceTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a simple interval job: every day at 09:30
	h := 9
	m := 30
	val := 1
	unit := "days"
	job := &Job{
		Name:               "Daily Job",
		SiteID:             1,
		CategoryID:         1,
		PromptID:           1,
		AIProviderID:       1,
		AIModel:            "gpt-4o",
		RequiresValidation: false,
		ScheduleType:       ScheduleInterval,
		IntervalValue:      &val,
		IntervalUnit:       &unit,
		ScheduleHour:       &h,
		ScheduleMinute:     &m,
		Status:             StatusActive,
	}

	err := svc.CreateJob(ctx, job)
	if err != nil {
		t.Fatalf("CreateJob error: %v", err)
	}

	// List and Get
	jobs, err := svc.ListJobs(ctx)
	if err != nil {
		t.Fatalf("ListJobs error: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}

	got, err := svc.GetJob(ctx, jobs[0].ID)
	if err != nil {
		t.Fatalf("GetJob error: %v", err)
	}
	if got.Name != "Daily Job" {
		t.Fatalf("unexpected name: %s", got.Name)
	}
	if got.NextRunAt == nil {
		t.Fatalf("NextRunAt should be set for active scheduled job")
	}
}

func TestJobService_ValidationErrors(t *testing.T) {
	svc, cleanup := setupJobServiceTest(t)
	defer cleanup()
	ctx := context.Background()

	bad := &Job{Name: "", SiteID: 1, CategoryID: 1, PromptID: 1, AIProviderID: 1, AIModel: "m", ScheduleType: ScheduleManual}
	if err := svc.CreateJob(ctx, bad); err == nil {
		t.Fatal("expected error for empty name")
	}

	bad2 := &Job{Name: "n", SiteID: 0, CategoryID: 1, PromptID: 1, AIProviderID: 1, AIModel: "m", ScheduleType: ScheduleManual}
	if err := svc.CreateJob(ctx, bad2); err == nil {
		t.Fatal("expected error for empty site")
	}

	// Interval with invalid unit
	val := 1
	unit := "years"
	bad3 := &Job{Name: "n", SiteID: 1, CategoryID: 1, PromptID: 1, AIProviderID: 1, AIModel: "m", ScheduleType: ScheduleInterval, IntervalValue: &val, IntervalUnit: &unit}
	if err := svc.CreateJob(ctx, bad3); err == nil {
		t.Fatal("expected error for invalid unit")
	}

	// Weekly without weekdays
	unitW := "weeks"
	bad4 := &Job{Name: "n", SiteID: 1, CategoryID: 1, PromptID: 1, AIProviderID: 1, AIModel: "m", ScheduleType: ScheduleInterval, IntervalValue: &val, IntervalUnit: &unitW}
	if err := svc.CreateJob(ctx, bad4); err == nil {
		t.Fatal("expected error for empty weekdays")
	}

	// Monthly without monthdays
	unitM := "months"
	bad5 := &Job{Name: "n", SiteID: 1, CategoryID: 1, PromptID: 1, AIProviderID: 1, AIModel: "m", ScheduleType: ScheduleInterval, IntervalValue: &val, IntervalUnit: &unitM}
	if err := svc.CreateJob(ctx, bad5); err == nil {
		t.Fatal("expected error for empty monthdays")
	}
}

func TestJobService_Pause_Resume_Update(t *testing.T) {
	svc, cleanup := setupJobServiceTest(t)
	defer cleanup()
	ctx := context.Background()

	h := 6
	m := 0
	val := 2
	unit := "days"
	job := &Job{
		Name:   "Every 2 days",
		SiteID: 1, CategoryID: 1, PromptID: 1,
		AIProviderID: 1, AIModel: "gpt-4o",
		ScheduleType: ScheduleInterval, IntervalValue: &val, IntervalUnit: &unit,
		ScheduleHour: &h, ScheduleMinute: &m,
		Status: StatusActive,
	}
	if err := svc.CreateJob(ctx, job); err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	jobs, _ := svc.ListJobs(ctx)
	jid := jobs[0].ID

	// Pause
	if err := svc.PauseJob(ctx, jid); err != nil {
		t.Fatalf("PauseJob: %v", err)
	}
	j, _ := svc.GetJob(ctx, jid)
	if j.Status != StatusPaused {
		t.Fatalf("expected paused, got %s", j.Status)
	}

	// Resume
	if err := svc.ResumeJob(ctx, jid); err != nil {
		t.Fatalf("ResumeJob: %v", err)
	}
	j, _ = svc.GetJob(ctx, jid)
	if j.Status != StatusActive {
		t.Fatalf("expected active, got %s", j.Status)
	}
	if j.NextRunAt == nil {
		t.Fatalf("NextRunAt should be set on resume")
	}

	// Update
	newH := 7
	job = j
	job.ScheduleHour = &newH
	if err := svc.UpdateJob(ctx, job); err != nil {
		t.Fatalf("UpdateJob: %v", err)
	}
	j2, _ := svc.GetJob(ctx, jid)
	if j2.ScheduleHour == nil || *j2.ScheduleHour != 7 {
		t.Fatalf("expected hour 7")
	}
}
