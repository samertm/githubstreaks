import $ from "jquery";
// Bootstrap requires jQuery to be global.
window.jQuery = window.$ = $;
// Require bootstrap from Bower. For some BS reason, "require" works
// but "import" doesn't (probably has to do with the module inheriting
// the global scope?)
require("../bower_components/bootstrap/dist/js/bootstrap");

import debuglog from "./debuglog";
import * as util from "./util";

// Refresh group commits button.
$("#refresh").click(function(e) {
  console.log("Refreshing group data.");
  var url = util.groupRefresh(util.getGID());
  $.ajax({
    type: "post",
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

// Enable tooltips.
$("[data-toggle='tooltip']").tooltip();

// Make #group-url select itself when you click on it.
$("#group-url").click(function(e) {
  $(this).select();
});

// createToggle returns a function for toggling a container on or
// off. Usage: createToggle(".day", ".all-repos")
function createToggle(parentSelector, containerSelector) {
  return function() {
    debuglog("toggle fired for parent '" + parentSelector + "', container '" + containerSelector + "'.");
    var container = $($(this).parents(parentSelector).find(containerSelector)[0]);
    if (container.is(":hidden")) {
      debuglog("sliding down");
      container.slideDown("slow");
    } else {
      debuglog("sliding up");
      container.slideUp("slow");
    }
  };
}

// Format commit groups.
(function() {
  var cgs = $("#commit-groups");
  if (!cgs) {
    return;
  }
  var days = cgs.find(".day");
  // If you see more than one day, collapse them!
  for (var i = 0; i < days.length; i++) {
    var day = days[i];
    if (i !== 0) {
      // Collapse the inner tags for every day but the first!
      $(day).find(".all-repos").hide();
    }
  }
  // Also, attach links to the days.
  $(".day-link").click(createToggle(".day", ".all-repos"));
  $(".repo-link").click(createToggle(".repo", ".all-commits"));
})();

