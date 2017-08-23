$(document).ready(function(){
  $("#save").click(function(){
    console.log("Save button clicked");
    conf = readConfiguration($("#dashboard"));
    console.log("Uploading configuration "+conf);
    uploadConfiguration(conf);
  });
  $('#getlocation').click(function(){
    console.log("Get location button clicked");
    getGeolocation($(this).parents(".location"));
  });
});

function getGeolocation(target) {
  if (navigator.geolocation) {
        navigator.geolocation.getCurrentPosition(function(position) {
          console.log($(target).find("#latitude"));
          $(target).find("#latitude").val(position.coords.latitude);
          $(target).find("#longitude").val(position.coords.longitude);
        });
  } else {
    $(target).find("#latitude") = "Geolocation is not supported by this browser.";
    $(target).find("#longitude") = "Geolocation is not supported by this browser.";
  }
}

function uploadConfiguration(configuration) {
  $.ajax({
    url: "/configuration",
    type: 'PUT',
    data: JSON.stringify(configuration),
    contentType: 'application/json',
    success: function(result) {
      if (result == "success") {
        $("#message").append('<div class="alert alert-success alert-dismissable"><a href="#" class="close" data-dismiss="alert" aria-label="close">&times;</a><strong>Configuration saved.</strong> Changes will take effect after a restart.</div>');
      } else {
        console.log(result);
      }
    }
  });
}

function readConfiguration(target){
  var bridge = Object();
  bridge.IP = $(target).find("#ip").val().trim();
  bridge.Username = $(target).find("#username").val().trim();

  var location = Object();
  location.Latitude = parseFloat($(target).find("#latitude").val().trim());
  location.Longitude = parseFloat($(target).find("#longitude").val().trim());

  var webinterface = Object();
  webinterface.enabled = $(target).find("#webinterfaceenabled").is(":checked");
  webinterface.port = parseInt($(target).find("#port").val().trim());

  var configuration = Object();
  configuration.Bridge = bridge
  configuration.Location = location
  configuration.WebInterface = webinterface

  return configuration;
}
