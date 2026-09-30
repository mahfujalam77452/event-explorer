{{template "partials/header.tpl" .}}

<section class="hero">
  <h1>Find events in your city</h1>
  <p class="muted">Search a city to browse Music and Sports events.</p>

  <form id="search-form" class="search-form" action="/events" method="get" autocomplete="off">
    <input
      type="text"
      id="city-input"
      placeholder="Start typing a city, e.g. Toronto"
      aria-label="City"
      role="combobox"
      aria-autocomplete="list"
      aria-expanded="false"
      aria-controls="suggestions"
      maxlength="100"
      required>

    <input type="hidden" name="city" id="city-value">
    <input type="hidden" name="countryCode" id="country-value">

    <button type="submit" id="search-btn" class="btn btn-primary" disabled>Search</button>

    <ul id="suggestions" class="suggestions" role="listbox" hidden></ul>
  </form>

  <p id="form-message" class="alert alert-error" role="alert" hidden></p>
  <p class="attribution">Powered by Google</p>

  <noscript>
    <p class="alert alert-error">City search needs JavaScript. Please enable it to continue.</p>
  </noscript>
</section>

<script src="/static/js/autocomplete.js" defer></script>

{{template "partials/footer.tpl" .}}