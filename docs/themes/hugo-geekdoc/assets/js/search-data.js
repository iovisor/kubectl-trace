'use strict';

(function() {
  const indexCfg = {{ with .Scratch.Get "geekdocSearchConfig" }}
    {{ . | jsonify}};
  {{ else }}
   {};
  {{ end }}

  indexCfg.doc = {
    id: 'id',
    field: ['title', 'content'],
    store: ['title', 'href'],
  };

  const index = FlexSearch.create(indexCfg);
  window.geekdocSearchIndex = index;

  {{ range $index, $page := .Site.Pages }}
  index.add({
    'id': {{ $index }},
    'href': '{{ $page.RelPermalink }}',
    'title': {{ (partial "title" $page) | jsonify }},
    'content': {{ $page.Plain | jsonify }}
  });
  {{- end -}}
})();
