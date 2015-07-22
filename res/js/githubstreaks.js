// SAMER: Idiomatic way of doing this?
// SAMER: Figure out how to use modules :V

var debug = true;

function debugLog() {
  if (debug) {
    console.log.apply(console, arguments);
  }
}

function fixFooterPosition(){
  var height = window.innerHeight;
  var container = document.getElementById("content");
  var footerHeight = document.getElementById("footer").clientHeight;
  if( container.clientHeight + footerHeight > height ){
    container.style.paddingBottom = "0px";
  }else{
    container.style.paddingBottom = (height-container.clientHeight-footerHeight) + "px";
  }
}

// Routes

function getGID() {
  var u = new URL(document.URL);
  var re = new RegExp("/group/(\\d+)")
  var match = re.exec(u.pathname)
  return Number(match[1])
}

window.onload = function() {
  debugLog("Window loaded.");
  fixFooterPosition();
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

  $('[data-toggle="tooltip"]').tooltip()
  $("#group-url").click(function(e) {
      $(this).select();
  });

  // Format commit groups.
  // BLAH @_@
  function formatCommitGroups() {
    var cgs = $("#commit-groups");
    if (!cgs) {
      return;
    }
    var days = cgs.find(".day")
    // If you see more than one day, collapse them!
    for (var i = 0; i < days.length; i++) {
      var day = days[i];
      if (i != 0) {
        // Collapse the inner tags for every day but the first!
        $(day).find(".repo").addClass("collapse");
      }
    }
    // Also, attach links to the days (eventually).
    $(".day-link").click(toggleDay);
    $(".repo-link").click(toggleRepo);
  }
  formatCommitGroups();

  // Toggle a single day on or off. Toggles off if any repo under the
  // day is collapsed.
  function toggleDay() {
    debugLog("toggleDay fired.");
    var repos = $(this).parents(".day").find(".repo");
    if (repos.hasClass("collapse")) {
      repos.removeClass("collapse");
    } else {
      repos.addClass("collapse");
    }
  }

  // Toggle a single repo on or off. Toggles off if any commit under
  // the repo is collapsed.
  function toggleRepo() {
    debugLog("toggleRepo fired.");
    var commits = $(this).parents(".repo").find(".commit");
    if (commits.hasClass("collapse")) {
      commits.removeClass("collapse");
    } else {
      commits.addClass("collapse");
    }
  }
}
