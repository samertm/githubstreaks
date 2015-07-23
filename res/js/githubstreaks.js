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

$(document).ready(function() {
  debugLog("Document ready.");
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

  // Set changes tags.
  var Changes = React.createClass({
    render: function() {
      return <span className="changes">
        <span className="additions">+ {this.props.additions}</span>
        <span> / </span>
        <span className="deletions">- {this.props.deletions}</span>
        </span>;
    },
  });
  $('[data-component="changes"]').each(function(i, e) {
    React.render(
        <Changes
      additions={e.getAttribute("data-additions")}
      deletions={e.getAttribute("data-deletions")}/>,
      e
    );
  });
  var DayBar = React.createClass({
    render: function() {
      var today = moment();
      var day = moment(this.props.day, "YYYY-MM-DD");
      var dayText;
      if (today.year() == day.year() &&
          today.dayOfYear() == day.dayOfYear()) {
        dayText = "today";
      } else {
        dayText = day.fromNow();
      }
      return <span className="day-link">{dayText}</span>;
    },
  });
  $('[data-component="day-bar"]').each(function(i, e) {
    React.render(
        <DayBar
      day={e.getAttribute("data-day")}/>,
      e
    );
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
        $(day).find(".all-repos").hide();
      }
    }
    // Also, attach links to the days.
    $(".day-link").click(createToggle(".day", ".all-repos"));
    $(".repo-link").click(createToggle(".repo", ".all-commits"));
  }
  formatCommitGroups();

  // createToggle returns a function for toggling a container on or
  // off. Usage: createToggle(".day", ".all-repos")
  function createToggle(parentSelector, containerSelector) {
    return function() {
      debugLog("toggle fired for parent '" + parentSelector + "', container '" + containerSelector +"'.");
      var container = $($(this).parents(parentSelector).find(containerSelector)[0]);
      if (container.is(":hidden")) {
        debugLog("sliding down");
        container.slideDown("slow");
      } else {
        debugLog("sliding up");
        container.slideUp("slow");
      }
    }
  }
});
