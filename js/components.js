import React from "react";
import $ from "jquery";
import moment from "moment";

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

$("[data-component='changes']").each(function(i, e) {
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
    if (today.year() === day.year() &&
        today.dayOfYear() === day.dayOfYear()) {
      dayText = "today";
    } else {
      dayText = day.fromNow();
    }
    return <span className="day-link">{dayText}</span>;
  },
});

$("[data-component='day-bar']").each(function(i, e) {
  React.render(
      <DayBar
    day={e.getAttribute("data-day")}/>,
    e
  );
});

