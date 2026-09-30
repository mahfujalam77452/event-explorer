(function () {
  'use strict';

  var DEBOUNCE_MS = 300;
  var MIN_CHARS = 3;

  var form = document.getElementById('search-form');
  var input = document.getElementById('city-input');
  var cityField = document.getElementById('city-value');
  var countryField = document.getElementById('country-value');
  var button = document.getElementById('search-btn');
  var list = document.getElementById('suggestions');
  var message = document.getElementById('form-message');

  if (!form || !input || !list) return;

  var sessionToken = newToken();
  var debounceTimer = null;
  var controller = null;   // cencel running request
  var requestId = 0;       // discurd previous response
  var items = [];
  var activeIndex = -1;

  // ---------- session token ----------

  function newToken() {
    if (window.crypto && crypto.randomUUID) return crypto.randomUUID();
    var chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
    var out = '';
    for (var i = 0; i < 32; i++) out += chars.charAt(Math.floor(Math.random() * chars.length));
    return out;
  }

  // filtered 

  function clearSelection() {
    cityField.value = '';
    countryField.value = '';
    button.disabled = true;
  }

  function setSelection(city, countryCode) {
    cityField.value = city;
    countryField.value = countryCode;
    button.disabled = false;
  }

  function showMessage(text) {
    message.textContent = text;
    message.hidden = false;
  }

  function hideMessage() {
    message.hidden = true;
    message.textContent = '';
  }

  // creating list

  function closeList() {
    list.hidden = true;
    list.innerHTML = '';
    items = [];
    activeIndex = -1;
    input.setAttribute('aria-expanded', 'false');
  }

  function showStatus(text) {
    list.innerHTML = '';
    var li = document.createElement('li');
    li.className = 'status';
    li.setAttribute('role', 'presentation');
    li.textContent = text;
    list.appendChild(li);
    list.hidden = false;
    items = [];
    activeIndex = -1;
  }

  function renderSuggestions(suggestions) {
    list.innerHTML = '';
    items = suggestions;
    activeIndex = -1;

    if (!suggestions.length) {
      showStatus('No cities found');
      return;
    }

        suggestions.forEach(function (s, i) {
      var li = document.createElement('li');
      li.setAttribute('role', 'option');
      li.dataset.index = String(i);
      li.textContent = s.text;

      li.addEventListener('mousedown', function (e) {
        e.preventDefault(); // holding input focus
        selectSuggestion(i);
      });
      list.appendChild(li);
    });

    list.hidden = false;
    input.setAttribute('aria-expanded', 'true');
  }

  function highlight(index) {
    var nodes = list.querySelectorAll('li[role="option"]');
    nodes.forEach(function (n) { n.classList.remove('active'); });
    if (index >= 0 && nodes[index]) {
      nodes[index].classList.add('active');
      nodes[index].scrollIntoView({ block: 'nearest' });
    }
    activeIndex = index;
  }

  // Calling server

  function fetchJSON(url, signal) {
    return fetch(url, { signal: signal, headers: { 'Accept': 'application/json' } })
      .then(function (res) {
        return res.json().catch(function () { return {}; }).then(function (data) {
          if (!res.ok) {
            throw new Error(data.error || 'Something went wrong. Please try again.');
          }
          return data;
        });
      });
  }

  function search(text) {
    if (controller) controller.abort();
    controller = new AbortController();
    var myId = ++requestId;

    showStatus('Searching...');

    var url = '/api/locations/autocomplete?input=' + encodeURIComponent(text) +
      '&sessionToken=' + encodeURIComponent(sessionToken);

    fetchJSON(url, controller.signal)
      .then(function (data) {
        if (myId !== requestId) return; // পুরনো উত্তর
        renderSuggestions(data.suggestions || []);
      })
      .catch(function (err) {
        if (err.name === 'AbortError' || myId !== requestId) return;
        showStatus(err.message);
      });
  }

  function selectSuggestion(index) {
    var s = items[index];
    if (!s) return;

    if (controller) controller.abort();
    controller = new AbortController();
    var myId = ++requestId;

    hideMessage();
    input.value = s.text;
    showStatus('Loading city...');

    var url = '/api/locations/' + encodeURIComponent(s.placeId) +
      '?sessionToken=' + encodeURIComponent(sessionToken);

        fetchJSON(url, controller.signal)
      .then(function (city) {
        if (myId !== requestId) return;
        input.value = city.label || s.text;
        setSelection(city.city, city.countryCode);
        closeList();
        // filtered done new token for new search
        sessionToken = newToken();
        button.focus();
      })
      .catch(function (err) {
        if (err.name === 'AbortError' || myId !== requestId) return;
        clearSelection();
        closeList();
        showMessage(err.message);
      });
  }

  // event

  input.addEventListener('input', function () {
    clearSelection(); 
    hideMessage();
    clearTimeout(debounceTimer);

    var text = input.value.trim();
    if (text.length < MIN_CHARS) {
      if (controller) controller.abort();
      requestId++;
      closeList();
      return;
    }
    debounceTimer = setTimeout(function () { search(text); }, DEBOUNCE_MS);
  });

  input.addEventListener('keydown', function (e) {
    if (list.hidden || !items.length) return;

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      highlight((activeIndex + 1) % items.length);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      highlight((activeIndex - 1 + items.length) % items.length);
    } else if (e.key === 'Enter' && activeIndex >= 0) {
      e.preventDefault();
      selectSuggestion(activeIndex);
    } else if (e.key === 'Escape') {
      closeList();
    }
  });

  input.addEventListener('blur', function () {
    setTimeout(closeList, 150);
  });

  // Preventing submission without selecting option
  form.addEventListener('submit', function (e) {
    if (!cityField.value || !countryField.value) {
      e.preventDefault();
      showMessage('Please choose a city from the suggestions.');
      input.focus();
    }
  });
})();