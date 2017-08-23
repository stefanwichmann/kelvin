$(document).ready(function(){
  $("#save").click(function(){
    console.log("Save button clicked");
    var schedules = new Array();
    $(".schedule").each(function(index) {
      schedules.push(readSchedule($(this)));
    });
    console.log(JSON.stringify(schedules));
    uploadSchedules(schedules)
  });
  $("#addSchedule").click(function(){
    console.log("Add schedule button clicked");
    addSchedule($("#schedules"))
  });
  $('#schedules').on('click', '.addEntryButton', function(){
    console.log("Add entry button clicked");
    addScheduleEntry($(this).parents("div.subschedule").find(".table"));
  });
  $('#schedules').on('click', '.deleteEntryButton', function(){
    console.log("Delete entry button clicked");
    $(this).parents("tr.entry").remove();
  });
  $('#schedules').on('click', '.testEntryButton', function(){
    console.log("Test entry button clicked");
    activateEntry($(this).parents("tr.entry"));
  });
  $('#schedules').on('click', '.deleteScheduleButton', function(){
    console.log("Delete schedule button clicked");
    $(this).parents("div.schedule").remove();
  });
});

function uploadSchedules(schedules) {
  $.ajax({
    url: "/schedules",
    type: 'PUT',
    data: JSON.stringify(schedules),
    contentType: 'application/json',
    success: function(result) {
      if (result == "success") {
        $("#message").append('<div class="alert alert-success alert-dismissable"><a href="#" class="close" data-dismiss="alert" aria-label="close">&times;</a><strong>Saved</strong> schedules.</div>');
      }
    }
  });
}

function readSchedule(target){
  var schedule = Object();
  schedule.beforeSunrise = readScheduleEntry($(target).find(".beforeSunrise"));
  schedule.afterSunset = readScheduleEntry($(target).find(".afterSunset"));
  schedule.defaultColorTemperature = parseInt($(target).find(".default .entry .colorTemperature").val().trim());
  schedule.defaultBrightness = parseInt($(target).find(".default .entry .brightness").val().trim());
  schedule.name = $(target).find(".name").val().trim();
  console.log($(target).find(".lights").val())
  schedule.associatedDeviceIDs = parseIDs($(target).find(".lights").val().trim());
  console.log(schedule);
  return schedule;
}

function readScheduleEntry(target) {
  var list = new Array();
  $(target).find(".entry").each(function(index) {
    var schedule = Object();
    schedule.time = $(this).find(".time").val().trim();
    schedule.colorTemperature = parseInt($(this).find(".colorTemperature").val().trim());
    schedule.brightness = parseInt($(this).find(".brightness").val().trim());
    console.log(schedule);
    list.push(schedule);
  });
  return list
}

function addScheduleEntry(target) {
  var entry = $('<tr class="entry">');
  entry.append('<td><input type="time" name="time" class="time form-control" value="10:00" autocomplete="off"></td>');
  entry.append('<td><input type="number" name="colorTemperature" class="colorTemperature form-control" value="2750" min="0" max="6500" autocomplete="off"></td>');
  entry.append('<td><input type="range" name="brightness" class="brightness form-control" value="100" min="0" max="100" autocomplete="off"></td>');
  entry.append('<td><div class="btn-group"><button type="button" class="deleteEntryButton btn btn-primary">Delete</button><button type="button" class="testEntryButton btn btn-primary">Test</button></div></td>');
  $(target).append(entry);
}

function addSchedule(target) {
  var schedule = $('<div class="schedule row well">');
  var collumn = $('<div class="col-md-12">')
  var basic = $('<form class="form-horizontal">');
  basic.append('<div class="form-group"><label>Name:</label><input type="text" class="name form-control" placeholder="Livingroom" autocomplete="off"></div>');
  basic.append('<div class="form-group"><label>Lights:</label><input type="text" class="lights form-control" placeholder="1,2,3" autocomplete="off"></div>');
  collumn.append(basic)

  <!-- Schedule before sunrise -->
  var subschedule = $('<div class="subschedule">');
  subschedule.append('<h1>Morning <small>(00:00 - sunrise)</small></h1>');
  var tableBeforeSunrise = $('<table class="beforeSunrise table">');
  var tbody = $('<tbody>')
  tbody.append('<tr><th scope="col">Time</th><th scope="col">Color Temperature</th><th scope="col">Brightness</th><th scope="col">Control</th></tr>');
  addScheduleEntry(tbody);
  tableBeforeSunrise.append(tbody);
  subschedule.append(tableBeforeSunrise);
  subschedule.append('<div class="text-center"><button type="button" class="addEntryButton btn btn-primary">Add entry</button></div>');
  collumn.append(subschedule);

  <!-- Default schedule -->
  var subschedule = $('<div class="subschedule">');
  subschedule.append('<h1>Daylight <small>(sunrise - sunset)</small></h1>');
  var tableDefault = $('<table class="default table">');
  var tbody = $('<tbody>')
  tbody.append('<tr><th scope="col">Time</th><th scope="col">Color Temperature</th><th scope="col">Brightness</th><th scope="col">Control</th></tr>');
  var form = $('<tr class="entry">');
  form.append('<td><input type="text" name="time" class="text form-control" value="sunrise - sunset" disabled></td>');
  form.append('<td><input type="number" name="colorTemperature" class="colorTemperature form-control" value="2750" min="0" max="6500" autocomplete="off"></td>');
  form.append('<td><input type="range" name="brightness" class="brightness form-control" value="100" min="0" max="100" autocomplete="off"></td>');
  form.append('<td><div class="btn-group"><button type="button" class="deleteEntryButton btn btn-primary" disabled>Delete</button><button type="button" class="testEntryButton btn btn-primary">Test</button></div></td>');
  tbody.append(form)
  tableDefault.append(tbody)
  subschedule.append(tableDefault);
  collumn.append(subschedule)

  <!-- Schedule after sunset -->
  var subschedule = $('<div class="subschedule">');
  subschedule.append('<h1>Evening <small>(sunset - 23:59)</small></h1>');
  var tableAfterSunset = $('<table class="afterSunset table">');
  var tbody = $('<tbody>')
  tbody.append('<tr><th scope="col">Time</th><th scope="col">Color Temperature</th><th scope="col">Brightness</th><th scope="col">Control</th></tr>');
  addScheduleEntry(tbody);
  tableAfterSunset.append(tbody)
  subschedule.append(tableAfterSunset);
  subschedule.append('<div class="text-center"><button type="button" class="addEntryButton btn btn-primary">Add entry</button></div>');
  collumn.append(subschedule);

  collumn.append('<div class="text-right"><button type="button" class="deleteScheduleButton btn btn-danger">Delete schedule</button></div>');
  schedule.append(collumn)
  target.append(schedule);
}

function activateEntry(target) {
  var entry = Object();
  entry.colorTemperature = parseInt(target.find(".colorTemperature").val());
  entry.brightness = parseInt(target.find(".brightness").val());
  console.log(JSON.stringify(entry));
  var associatedDeviceIDs =  parseIDs(target.parents("div.schedule").find(".lights").val());
  for (i = 0; i < associatedDeviceIDs.length; i++) {
    $.ajax({
      url: "/lights/"+associatedDeviceIDs[i]+"/activate",
      type: 'PUT',
      data: JSON.stringify(entry),
      contentType: 'application/json'
    });
  }
}

function parseIDs(text) {
  if (text.trim() == "") {
    return Array();
  }
  var ids = text.trim().split(",");
  for (index in ids ) {
    ids[index] = parseInt(ids[index], 10);
  }
  return ids;
}
