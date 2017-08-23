$(document).ready(function(){
  $('#dashboard').on('click', '.enableKelvinButton', function(){
    console.log("Activate kelvin button clicked");
    activateKelvin($(this).parents("tr.light"));
  });
  $('#dashboard').on('click', '#restartKelvinButton', function(){
    console.log("Restart kelvin button clicked");
    restartKelvin();
  });
  $.ajax({
    url: "/lights",
    type: 'GET',
    success: function(result) {
      $.each(result, function( index, value ) {
        console.log( index + ": " + value );
      });
    }
  });
});

function updateLight(light) {
  var yes = '<td><span class="glyphicon glyphicon-ok-circle text-success"></span></td>'
  var no = '<td><span class="glyphicon glyphicon-remove-circle text-danger"></span></td>'

  var entry = $('<tr class="light">');
  entry.append('<td class="id">' + light.ID + '</td>');
  entry.append('<td>' + light.Name + '</td>');
  if (light.Reachable) { entry.append(yes); } else { entry.append(no); }
  if (light.On) { entry.append(yes); } else { entry.append(no); }
  if (light.Tracking) { entry.append(yes); } else { entry.append(no); }
  if (light.Automatic) { entry.append(yes); } else { entry.append(no); }
  if (light.Automatic) {
    entry.append('<td>' + light.CurrentLightState.ColorTemperature + '<small>K</small></td>');
  } else {
     entry.append('<td>' + light.TargetLightState.ColorTemperature + '<small>K</small></td>');
  }
  entry.append('<td>' + light.CurrentLightState.Brightness + '<small>%</small></td>');
  console.log(entry);
}

function activateKelvin(entry) {
  console.log($(entry).find(".id").html());
  $.ajax({
    url: "/lights/"+ $(entry).find(".id").html() +"/automatic",
    type: 'PUT'
  });
  $(entry).find(".enableKelvinButton").prop("disabled",true);
  window.setTimeout(function(){location.reload(true);}, 2000);
}

function restartKelvin() {
  $.ajax({
    url: "/restart",
    type: 'PUT'
  });
  $("#restartKelvinButton").prop("disabled",true);
  $("#message").append('<div class="alert alert-success alert-dismissable"><a href="#" class="close" data-dismiss="alert" aria-label="close">&times;</a><strong>Restarting... Please wait</div>');
  window.setTimeout(function(){location.reload(true);}, 5000);
}
