package storage

import "path/filepath"

type ReleaseLayout struct {
	root string
}

func NewReleaseLayout(root string) *ReleaseLayout {
	return &ReleaseLayout{root: root}
}

func (r *ReleaseLayout) ProjectDir(projectID string) string {
	return filepath.Join(r.root, projectID)
}

func (r *ReleaseLayout) UploadsDir(projectID string) string {
	return filepath.Join(r.ProjectDir(projectID), "uploads")
}

func (r *ReleaseLayout) RuntimeDir(projectID string) string {
	return filepath.Join(r.ProjectDir(projectID), "runtime")
}

func (r *ReleaseLayout) ReleasesDir(projectID string) string {
	return filepath.Join(r.ProjectDir(projectID), "releases")
}

func (r *ReleaseLayout) ReleaseDir(projectID, releaseID string) string {
	return filepath.Join(r.ReleasesDir(projectID), releaseID)
}

func (r *ReleaseLayout) PublicDir(projectID, releaseID string) string {
	return filepath.Join(r.ReleaseDir(projectID, releaseID), "public")
}

func (r *ReleaseLayout) PublicRoot() string {
	return filepath.Join(filepath.Dir(r.root), "public")
}

func (r *ReleaseLayout) PreviewDir(jobID string) string {
	return filepath.Join(r.PublicRoot(), "_previews", jobID)
}

func (r *ReleaseLayout) LivePublicDir(username, slug string) string {
	return filepath.Join(r.PublicRoot(), "@"+username, slug)
}
