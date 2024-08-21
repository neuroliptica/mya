const form = document.getElementById("post-form");

function post() {
  const data = new FormData(form);

  return fetch("http://127.0.0.1:3000/api/post", {
    method: "post",
    body: data,
  })
    .then((r) => r.json())
    .then((j) => {
      console.log(j);
      // reload page after response.
      location.reload();
    })
    .catch((e) => console.log(e));
}

form.addEventListener("submit", (e) => {
  e.preventDefault();
  post();
})

