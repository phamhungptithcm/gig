(function () {
  function fallbackCopy(text) {
    var textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.setAttribute("readonly", "");
    textarea.style.position = "absolute";
    textarea.style.left = "-9999px";
    document.body.appendChild(textarea);
    textarea.select();

    try {
      document.execCommand("copy");
    } finally {
      document.body.removeChild(textarea);
    }
  }

  function copyText(text) {
    if (navigator.clipboard && window.isSecureContext) {
      return navigator.clipboard.writeText(text);
    }

    return new Promise(function (resolve) {
      fallbackCopy(text);
      resolve();
    });
  }

  function addCopyButtons() {
    var blocks = document.querySelectorAll("pre > code");

    blocks.forEach(function (code) {
      var pre = code.parentElement;
      if (!pre || pre.parentElement.classList.contains("gig-code-block")) {
        return;
      }

      var wrapper = document.createElement("div");
      wrapper.className = "gig-code-block";
      pre.parentNode.insertBefore(wrapper, pre);
      wrapper.appendChild(pre);

      var button = document.createElement("button");
      button.type = "button";
      button.className = "gig-copy-button";
      button.textContent = "Copy";
      button.setAttribute("aria-label", "Copy code block");

      button.addEventListener("click", function () {
        copyText(code.innerText).then(function () {
          button.textContent = "Copied";
          button.classList.add("is-copied");

          window.setTimeout(function () {
            button.textContent = "Copy";
            button.classList.remove("is-copied");
          }, 1500);
        });
      });

      wrapper.appendChild(button);
    });
  }

  document.addEventListener("DOMContentLoaded", addCopyButtons);
})();
