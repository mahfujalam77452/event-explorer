{{template "partials/header.tpl" .}}

<section class="error-page">
  <h1>{{.Heading}}</h1>
  <p class="muted">{{.Message}}</p>
  <a class="btn btn-primary" href="/">Go to home</a>
</section>

{{template "partials/footer.tpl" .}}