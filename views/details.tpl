{{template "partials/header.tpl" .}}

<a class="back-link" href="{{.BackURL}}">&larr; {{.BackLabel}}</a>

{{if .PageError}}
  <div class="alert alert-error">{{.PageError}}</div>
  <a class="btn btn-primary" href="/">Search a city</a>
{{else}}
  {{with .Event}}
  <article class="details">
    {{if .ImageURL}}
      <img class="details-img" src="{{.ImageURL}}" alt="{{.Name}}">
    {{else}}
      <div class="details-img placeholder" style="height:240px">No image</div>
    {{end}}

    <div class="details-body">
      <h1>{{.Name}}</h1>

      <p class="meta"><strong>Date:</strong> {{if .Date}}{{.Date}}{{else}}Date to be announced{{end}}</p>
      <p class="meta"><strong>Location:</strong> {{if .Location}}{{.Location}}{{else}}Location to be announced{{end}}</p>

      {{if .Description}}
        <h2>About this event</h2>
        <p class="description">{{.Description}}</p>
      {{end}}

      <div class="details-actions">
        {{if .TicketURL}}
          <a class="btn btn-primary" href="/redirect/{{.ID}}" rel="nofollow">View Tickets</a>
        {{else}}
          <p class="alert alert-empty">Tickets are not available for this event yet.</p>
        {{end}}
      </div>
    </div>
  </article>
  {{end}}
{{end}}

{{template "partials/footer.tpl" .}}