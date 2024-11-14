// TODO: Take zoom from coverage responses.
const zoom = 20;

const mapOptions = {
  fadeAnimation: false,
};
const map = L.map('map', mapOptions).setView([-33.888, 151.16], 20);
map.doubleClickZoom.disable();

L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
  maxZoom: zoom,
  attribution: '&copy; <a href="http://www.openstreetmap.org/copyright">OpenStreetMap</a>'
}).addTo(map);

var date = undefined;
var dates = [];

// TODO: Add explanation of this data structure.
var vertLayers = [];

function pickDate(newDate) {
  if (date == newDate) {
    return;
  }
  date = newDate;

  var idx = dates.indexOf(date);
  if (idx == -1) {
    console.log("ERROR: selected date not in coverage:", date, dates);
  }

  document.getElementById("first").disabled = false;
  document.getElementById("prev").disabled = idx == dates.length - 1;
  document.getElementById("next").disabled = idx == 0;
  document.getElementById("last").disabled = false;

  document.getElementById("date").innerHTML = date;

  // TODO: Use maxNativeZoom and minNativeZoom options.
  const layer = L.tileLayer(`/tiles/{z}/{x}/{y}?date=${date}`, {
    maxZoom: 21,
    attribution: '&copy; <a href="https://www.nearmap.com">Nearmap</a>',
    detectRetina: false, // TODO: Enable retina support? (needs zoom adjustment?)
  });
  layer.addTo(map);

  vertLayers.push(layer); // Add to end of array.

  // TODO: The 'load' event is when the images are loaded from the network, NOT
  // when they are loaded onto the screen. So this approach ends up removing
  // the layer before the new layer is completely displayed.
  //
  // Idea for a work around: rather than removing the layer, instead set the
  // URL to a blank image. Remove layers that are more than 1 or 2 layers away
  // from the current layer as a buffer.
  layer.on('load', function() {
    while (vertLayers[0] != layer) {
      map.removeLayer(vertLayers[0]);
      vertLayers.shift()
    }
  });

  // TODO: Change the date on the download popup.
}

function clearDate() {
  if (date == undefined) {
    return;
  }
  date = undefined;

  document.getElementById("first").disabled = true;
  document.getElementById("prev").disabled = true;
  document.getElementById("next").disabled = true;
  document.getElementById("last").disabled = true;

  document.getElementById("date").innerHTML = "no imagery";
}

document.getElementById("first").addEventListener('click', function() {
  pickDate(dates[dates.length - 1]); // TODO: Check for out of bounds? Disable button?
});

document.getElementById("prev").addEventListener('click', function() {
  var idx = dates.indexOf(date);
  idx++; // TODO: Check for out of bounds? Disable button?
  pickDate(dates[idx]);
});

document.getElementById("next").addEventListener('click', function() {
  var idx = dates.indexOf(date);
  idx--; // TODO: Check for out of bounds? Disable button?
  pickDate(dates[idx]);
});

document.getElementById("last").addEventListener('click', function() {
  pickDate(dates[0]); // TODO: Check for out of bounds? Disable button?
});

function updateCoverage() {
  const bounds = map.getBounds();
  const xhr = new XMLHttpRequest();
  const u = `/coverage?minlon=${bounds.getWest()}&minlat=${bounds.getSouth()}&maxlon=${bounds.getEast()}&maxlat=${bounds.getNorth()}`;
  xhr.open('GET', u, false);
  xhr.send();
  dates = JSON.parse(xhr.responseText);

  if (dates.length == 0) {
    clearDate();
    return;
  }

  if (date == undefined) {
    pickDate(dates[0]);
    return;
  }

  // Find the date in the new list of dates that is closest to the current
  // date (which might be the same date).
  var closest = Infinity;
  for (var i = 0; i < dates.length; i++) {
    var diff = Math.abs(new Date(dates[i]) - new Date(date));
    if (diff < closest) {
      closest = diff;
      idx = i;
    }
  }
  pickDate(dates[idx]);
}

updateCoverage();

// TODO: When zooming in and out with a scroll wheel, the rectangle sometimes
// pops in and out in size.

const cursorRectLayer = L.rectangle([[0, 0], [0, 0]], {
  color: 'lightblue',
  weight: 2,
  lineCap: 'square',
  lineJoin: 'square',
  fill: false,
});
cursorRectLayer.addTo(map);

map.on('mousemove', function(e) {
  updateCursorRect(e.latlng);
});

function updateCursorRect(centre) {
  const xy = map.project(centre, zoom);
  const nw = xy.subtract({x: xres / 2, y: yres / 2});
  const se = nw.add({x: xres, y: yres});
  const bounds = L.latLngBounds(
    map.unproject(nw, zoom),
    map.unproject(se, zoom),
  );
  cursorRectLayer.setBounds(bounds);
}

var screenshotRectLayer = L.rectangle([[0, 0], [0, 0]], {color: 'blue'});

function zeroPad(num, places) {
  return String(num).padStart(places, '0')
}

map.on('click', function(e) {
  screenshotRectLayer.setBounds(cursorRectLayer.getBounds());
  screenshotRectLayer.addTo(map);

  const nw = screenshotRectLayer.getBounds().getNorthWest();
  const xy = map.project(nw, zoom);
  const x = Math.round(xy.x);
  const y = Math.round(xy.y);

  const d = new Date();
  const date_str = `${d.getFullYear()}-${zeroPad(d.getMonth() + 1, 2)}-${zeroPad(d.getDate(), 2)}`;
  const center = screenshotRectLayer.getBounds().getCenter()
  const location_str = `${center.lat.toFixed(5)}_${center.lng.toFixed(5)}`;
  const name = `${date_str}_${xres/downsample}x${yres/downsample}_${location_str}_screenshot.jpg`; // TODO: add multiplier, etc.
  // TODO: Build this URL properly, rather than with string interpolation.
  const url = `/download?x=${x}&y=${y}&xres=${xres}&yres=${yres}&zoom=${zoom}&date=${date}&downsample=${downsample}`;
  const anchor = `<a download="${name}" href="${url}">Download</a>`;

  const p = L.popup(screenshotRectLayer.getCenter(), {content: anchor});
  p.on('remove', function() {
    map.removeLayer(screenshotRectLayer);
  });
  p.openOn(map);
});

map.on('moveend', function(e) {
  updateCoverage();
});

var xres;
var yres;
var downsample;

const resolutionSelect = document.getElementById("resolution");
const multiplierSelect = document.getElementById("multiplier");

function onResolutionOrMultiplierChange() {
  const pair = resolutionSelect.value.split("x").map(Number);
  xres = pair[0];
  yres = pair[1];

  const mult = Number(multiplierSelect.value);
  xres *= mult;
  yres *= mult;
  downsample = mult;

  updateCursorRect(cursorRectLayer.getBounds().getCenter());
}

// Pull the initial resolution and multiplier from the select elements.
onResolutionOrMultiplierChange();

resolutionSelect.addEventListener("input", onResolutionOrMultiplierChange);
multiplierSelect.addEventListener("input", onResolutionOrMultiplierChange);
