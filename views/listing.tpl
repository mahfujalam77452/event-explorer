{{template "partials/header.tpl" .}}

<a class="back-link" href="/">&larr; Change city</a>

{{if .PageError}}
  <div class="alert alert-error">{{.PageError}}</div>
  <a class="btn btn-primary" href="/">Search a city</a>
{{else}}
  <h1>Events in {{.City}}, {{.CountryCode}}</h1>

  {{range .Sections}}
  <section class="section">
    <h2 class="section-title">{{.Category}}</h2>

    {{if .Error}}
      <div class="alert alert-error">{{.Error}}</div>
    {{else if .Events}}
      <div class="grid">
        {{range .Events}}
        <article class="card">
          {{if .ImageURL}}
            <img class="card-img" src="{{.ImageURL}}" alt="{{.Name}}" loading="lazy">
          {{else}}
            <div class="card-img placeholder">No image</div>
          {{end}}
          <div class="card-body">
            <h3>{{.Name}}</h3>
            <p class="meta">{{if .Date}}{{.Date}}{{else}}Date to be announced{{end}}</p>
            <p class="meta">{{if .Venue}}{{.Venue}}{{else}}Venue to be announced{{end}}</p>
            <a class="btn btn-primary" href="/events/{{.ID}}?city={{$.City}}&countryCode={{$.CountryCode}}">View Details</a>
          </div>
        </article>
        {{end}}
      </div>
    {{else}}
      <div class="alert alert-empty">No {{.Category}} events found in {{$.City}}.</div>
    {{end}}
  </section>
  {{end}}
{{end}}

{{template "partials/footer.tpl" .}}