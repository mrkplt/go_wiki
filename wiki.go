package main

import (
  "html/template"
  "net/http"
  "regexp"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
)

type Page struct {
  Title string
  Body  []byte
}

var templates = template.Must(template.ParseFiles("templates/edit.html", "templates/view.html"))
var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

func (p *Page) save() error {
  session, err := mgo.Dial("localhost")
  if err != nil {
    panic(err)
  }
  defer session.Close()

  session.SetMode(mgo.Monotonic, true)

  c := session.DB("wiki").C("Pages")

  err = c.Update(bson.M{"title": p.Title}, &Page{Title: p.Title, Body: p.Body})
  if err != nil {
    err = c.Insert(&Page{Title: p.Title, Body: p.Body})
    if err != nil {
      return err
    }
  }

   return err
}

func loadPage(title string) (*Page, error) {
  session, err := mgo.Dial("localhost")
  if err != nil {
    panic(err)
  }
  defer session.Close()

  session.SetMode(mgo.Monotonic, true)

  c := session.DB("wiki").C("Pages")

  result := Page{}
  err = c.Find(bson.M{"title":title}).One(&result)
  if err != nil {
    return nil, err
  }
  return &Page{Title: title, Body: result.Body}, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
  p, err := loadPage(title)
  if err != nil {
    http.Redirect(w, r, "/edit/"+title, http.StatusFound)
    return
  }
  renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
  p, err := loadPage(title)
  if err != nil {
    p = &Page{Title: title}
  }
  renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
  body := r.FormValue("body")
  p := &Page{Title: title, Body: []byte(body)}
  err := p.save()
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page){
  err := templates.ExecuteTemplate(w, tmpl+".html", p)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
  return func(w http.ResponseWriter, r *http.Request){
    m := validPath.FindStringSubmatch(r.URL.Path)
    if m == nil {
        http.NotFound(w, r)
        return
    }
    fn(w, r, m[2])
  }
}

func main(){
  http.HandleFunc("/view/", makeHandler(viewHandler))
  http.HandleFunc("/edit/", makeHandler(editHandler))
  http.HandleFunc("/save/", makeHandler(saveHandler))
  http.ListenAndServe(":8080", nil)
}
