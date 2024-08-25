
// === Reply form ===
var reply = document.getElementById("new-reply");
var form = document.getElementById("post-form");

reply.addEventListener("click", () => {
  let f = document.getElementById("reply-form");
  if (f.style.display === "none") {
    loadCaptcha();
    f.style.display = "block";
    return;
  }
  f.style.display = "none";
})

function genCaptcha() {
  return fetch("http://127.0.0.1:3000/api/captcha/new")
    .then((r) => r.json())
    .then((r) => {
      response = {
        id: "",
      };
      if ("id" in r) {
        response = r;
      } else {
        console.log(r["error"]);
      }
      return response;
    })
}

function loadCaptcha() {
  return genCaptcha()
    .then((r) => {
      let g = document.getElementById("form-sage");

      // Replace an existing node or create new if not exists.
      let replace = (id, c) => {
        let t = document.getElementById(id);
        if (t !== null) {
          t.replaceWith(c);
          return;
        }
        g.appendChild(c);
      };

      // Append captcha_id input field to form.
      let c = document.createElement("input");
      c.setAttribute("id", "captcha_id");
      c.setAttribute("type", "hidden");
      c.setAttribute("name", "captcha_id");
      c.setAttribute("value", r["id"]);

      replace("captcha_id", c);

      // Load captcha image and append to form.
      let img = document.createElement("img");
      const src = "http://127.0.0.1:3000/api/captcha/get?id=" + r["id"];
      img.setAttribute("id", "captcha_image");
      img.setAttribute("src", src);
      img.setAttribute("width", "170");
      img.setAttribute("height", "25");
      img.setAttribute("onclick", "loadCaptcha();")

      replace("captcha_image", img);
    })
    .catch((e) => console.log(e));
}

function post() {
  const data = new FormData(form);

  return fetch("http://127.0.0.1:3000/api/post", {
    method: "post",
    body: data,
  })
    .then((r) => r.json())
    .then((j) => {
      console.log(j);
      // Reload current page after response.
      location.reload();
    })
    .catch((e) => console.log(e));
}

form.addEventListener("submit", (e) => {
  e.preventDefault();
  post();
})

