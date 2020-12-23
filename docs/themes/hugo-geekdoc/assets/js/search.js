'use strict';

{{ $searchDataFile := printf "js/%s.search-data.js" .Language.Lang }}
{{ $searchData := resources.Get "js/search-data.js" | resources.ExecuteAsTemplate $searchDataFile . | resources.Minify | resources.Fingerprint }}

(function() {
  const input = document.querySelector('#gdoc-search-input');
  const results = document.querySelector('#gdoc-search-results');

  input.addEventListener('focus', init);
  input.addEventListener('keyup', search);

  function init() {
    input.removeEventListener('focus', init); // init once
    input.required = true;

    loadScript('{{ "js/flexsearch.min.js" | relURL }}');
    loadScript('{{ $searchData.RelPermalink }}', function() {
      input.required = false;
      search();
    });
  }

  function search() {
    while (results.firstChild) {
      results.removeChild(results.firstChild);
    }

    if (!input.value) {
      console.log("empty")
      results.classList.remove("has-hits");
      return;
    }

    const searchHits = window.geekdocSearchIndex.search(input.value, 10);

    console.log(searchHits.length);
    if (searchHits.length > 0) {
      results.classList.add("has-hits");
    } else {
      results.classList.remove("has-hits");
    }

    searchHits.forEach(function(page) {
      const li = document.createElement('li'),
            a = li.appendChild(document.createElement('a'));

      a.href = page.href;
      a.textContent = page.title;

      results.appendChild(li);
      results.classList.add("DUMMY");
    });

  }

  function loadScript(src, callback) {
    const script = document.createElement('script');
    script.defer = true;
    script.async = false;
    script.src = src;
    script.onload = callback;

    document.head.appendChild(script);
  }
})();
