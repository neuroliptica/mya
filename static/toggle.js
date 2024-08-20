function changeVisible(id) {
  let f = document.getElementById(id);
  console.log(f.style.display);
  f.style.display = (f.style.display === "none" ? "block" : "none");
}
