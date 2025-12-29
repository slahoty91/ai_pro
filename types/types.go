package types

type FileType string

const (
	FileTypeResume       FileType = "resume"
	FileTypePrescription FileType = "prescription"
)

type SearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type DocumentUploadRequest struct {
	FileType FileType `form:"file_type" json:"file_type"`
	Title    string   `form:"title" json:"title"`
}

type ExperienceEntry struct {
	Company     string `json:"Company"`
	Duration    string `json:"Duration"`
	Position    string `json:"Position"`
	Description string `json:"Description"`
}

type EducationEntry struct {
	Institution string `json:"Institution"`
	Degree      string `json:"Degree,omitempty"`
	Location    string `json:"Location"`
}

type ProjectEntry struct {
	Name        string `json:"Name"`
	Description string `json:"Description"`
}

type ExtractedProfile struct {
	Name              string            `json:"Name"`
	Email             string            `json:"Email"`
	Phone             string            `json:"Phone"`
	Address           string            `json:"Address"`
	City              string            `json:"City"`
	State             string            `json:"State"`
	Zip               string            `json:"Zip"`
	Country           string            `json:"Country"`
	LinkedIn          string            `json:"LinkedIn"`
	Skills            []string          `json:"Skills"`
	MostRelevantSkill string            `json:"Most Relevant Skill"`
	Experience        []ExperienceEntry `json:"Experience"`
	Education         []EducationEntry  `json:"Education"`
	Projects          []ProjectEntry    `json:"Projects"`
	Certifications    []string          `json:"Certifications"`
	Publications      []string          `json:"Publications"`
	Companies         []string          `json:"Companies"`
	CurrentCompany    string            `json:"Current Company"`
	CurrentTitle      string            `json:"Current Title"`
	CurrentLocation   string            `json:"Current Location"`
}

type MedicationEntry struct {
	Name         string `json:"Name"`
	Dosage       string `json:"Dosage"`
	Frequency    string `json:"Frequency"`
	Duration     string `json:"Duration"`
	Instructions string `json:"Instructions"`
}

type ExtractedPrescription struct {
	PatientName    string            `json:"Patient Name"`
	PatientDOB     string            `json:"Patient DOB"`
	PatientAddress string            `json:"Patient Address"`
	DoctorName     string            `json:"Doctor Name"`
	DoctorLicense  string            `json:"Doctor License"`
	Date           string            `json:"Date"`
	PrescriptionID string            `json:"Prescription ID"`
	Medications    []MedicationEntry `json:"Medications"`
	Pharmacy       string            `json:"Pharmacy"`
	Instructions   string            `json:"Instructions"`
}
