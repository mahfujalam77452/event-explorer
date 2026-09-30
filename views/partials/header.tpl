<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <link rel="stylesheet" href="/static/css/style.css">
</head>
<body>
  <header class="site-header">
    <div class="container header-inner">
      <a class="brand" href="/">{{.SiteName}}</a>
      <nav>
        <a href="/" {{if eq .CurrentPath "/"}}class="active"{{end}}>Home</a>
      </nav>
    </div>
  </header>
  <main class="container">