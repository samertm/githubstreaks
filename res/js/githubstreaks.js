// SAMER: Idiomatic way of doing this?
// SAMER: Figure out how to use modules :V

// Routes

function getGID() {
  var u = new URL(document.URL);
  var re = new RegExp("/group/(\\d+)")
  var match = re.exec(u.pathname)
  return Number(match[1])
}

window.onload = function() {
  // Refresh group commits button.
  $('#refresh').click(function(e) {
    console.log("Refreshing group data.");
    var url = "/group/" + getGID() + "/refresh";
    $.ajax({
      type: "post", // SAMER: POST is correct, right?
      url: url,
      success: function() {
        // Reload the page.
        document.location.reload(true);
      },
      error: function(xhr, status, error) {
        console.log("ERROR REFRESING DATA: " + error);
      },
    });
  });
}
