package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"text/template"

	"github.com/docshelf/docshelf"
	"github.com/go-chi/chi"
	"github.com/russross/blackfriday"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

// A TagReq is a request to apply tags to a given document.
type TagReq struct {
	Path string
	Tags []string
}

// A DocHandler has methods that can handle HTTP requests for Docs.
type DocHandler struct {
	docStore docshelf.DocStore
	log      *logrus.Logger
}

// NewDocHandler returns a DocHandler struct using the given DocStore and Logger instance.
func NewDocHandler(docStore docshelf.DocStore, logger *logrus.Logger) DocHandler {
	return DocHandler{
		docStore: docStore,
		log:      logger,
	}
}

// PostDoc handles requests for posting new (or existing) Docs.
func (h DocHandler) PostDoc(w http.ResponseWriter, r *http.Request) {
	var doc docshelf.Doc
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		h.log.Error(err)
		badRequest(w, "invalid request body, could not save document")
		return
	}

	// need to make sure we grab author information from the user's session
	user, err := getContextUser(r.Context())
	if err != nil {
		h.log.Error(err)
		serverError(w, "something went wrong while determining author")
		return
	}

	doc.CreatedBy = user.ID
	doc.UpdatedBy = user.ID

	id, err := h.docStore.PutDoc(r.Context(), doc)
	if err != nil {
		h.log.Error(err)
		serverError(w, "something went wrong while saving document")
		return
	}

	data, err := json.Marshal(ID{id})
	if err != nil {
		h.log.Error(err)
		serverError(w, "something went wrong while returning ID")
		return
	}

	okJSON(w, data)
}

// PinDoc handles requests from users to pin a document. It applies a special tag to the doc
// which can be used later to find a user's pinned documents.
func (h DocHandler) PinDoc(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	user, err := getContextUser(r.Context())
	if err != nil {
		h.log.Error(err)
		serverError(w, "something went wrong while pinning document")
		return
	}

	if err := h.docStore.TagDoc(r.Context(), id, fmt.Sprintf("user/%s", user.ID)); err != nil {
		h.log.Error(err)
		serverError(w, "something went wrong while pinning document")
		return
	}

	noContent(w)
}

// GetList handles requests for listing Docs by path prefix.
func (h DocHandler) GetList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	tags := strings.Split(r.URL.Query().Get("tags"), ",")
	if len(tags) == 1 && tags[0] == "" {
		tags = nil
	}

	docs, err := h.docStore.ListDocs(r.Context(), query, tags...)
	if err != nil {
		h.log.Error(err)
		serverError(w, "something went wrong while listing documents")
		return
	}

	// don't return 'null' values for empty results
	if len(docs) == 0 {
		okJSON(w, []byte("[]"))
		return
	}

	data, err := json.Marshal(docs)
	if err != nil {
		h.log.Error(err)
		serverError(w, "something went wrong while serializing documents")
		return
	}

	okJSON(w, data)
}

// PostTag handles requests for posting tags to an existing Doc.
func (h DocHandler) PostTag(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var tags []string
	if err := json.NewDecoder(r.Body).Decode(&tags); err != nil {
		h.log.Error(err)
		badRequest(w, "invalid format for tagging documents")
		return
	}

	if err := h.docStore.TagDoc(r.Context(), id, tags...); err != nil {
		h.log.Error(err)
		serverError(w, "something went wrong while tagging document")
		return
	}

	noContent(w)
}

// GetDoc handles requests for fetching specific Docs.
func (h DocHandler) GetDoc(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	doc, err := h.docStore.GetDoc(r.Context(), id)
	if err != nil {
		if docshelf.CheckNotFound(err) {
			notFound(w)
			return
		}

		h.log.Error(err)
		serverError(w, "something went wrong while fetching document")
		return
	}

	data, err := json.Marshal(doc)
	if err != nil {
		h.log.Error(err)
		serverError(w, "something went wrong while serializing document")
		return
	}

	okJSON(w, data)
}

// DeleteDoc handles requests for removing specific Docs.
func (h DocHandler) DeleteDoc(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.docStore.RemoveDoc(r.Context(), id); err != nil {
		h.log.Error(err)
		serverError(w, "something went wrong while deleting document")
		return
	}

	noContent(w)
}

// RenderDoc handles requests for rendering Documents as HTML.
func (h DocHandler) RenderDoc(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "path")

	doc, err := h.docStore.GetDoc(r.Context(), path)
	if err != nil {
		log.Error(err)
		serverError(w, "could not render page")
		return
	}

	dom := blackfriday.Run([]byte(doc.Content))
	doc.Content = string(dom)

	// TODO (erik): Need to embed this template in the binary rather than reading off of
	// the file system.
	f, err := ioutil.ReadFile("./template.html")
	if err != nil {
		log.Error(err)
		serverError(w, "could not render page")
		return
	}

	tmpl, err := template.New("doc").Parse(string(f))
	if err != nil {
		log.Error(err)
		serverError(w, "could not render page")
		return
	}

	data := make([]byte, 0)
	output := bytes.NewBuffer(data)
	if err := tmpl.Execute(output, doc); err != nil {
		log.Error(err)
		serverError(w, "could not render page")
		return
	}

	okHTML(w, output.Bytes())
}
