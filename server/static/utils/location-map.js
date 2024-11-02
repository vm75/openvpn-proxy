export default {
  name: 'location-map',
  props: {
    latitude: {
      type: String,
      required: true
    },
    longitude: {
      type: String,
      required: true
    },
    city: {
      type: String,
      default: 'Unknown Location'
    }
  },
  data() {
    return {
      map: null
    }
  },
  template: `
    <div>
      <!-- Map Container -->
      <div id="vue-map" ref="map" style="height: 200px; border-radius: 5px;"></div>
    </div>
  `,
  methods: {
    initMap() {
      if (this.map) {
        this.map.remove();
      }

      // Initialize map
      this.map = L.map(this.$refs.map).setView([this.latitude, this.longitude], 13);

      // Load and add tile layer
      L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        maxZoom: 18,
        attribution: '© OpenStreetMap'
      }).addTo(this.map);

      // Add marker to map with popup
      L.marker([this.latitude, this.longitude]).addTo(this.map)
        .bindPopup(`VPN Server Location: ${this.city}`)
        .openPopup();
    },
  },
  watch: {
    latitude() {
      this.initMap();
    },
    longitude() {
      this.initMap();
    },
    city() {
      this.initMap();
    }
  },
  mounted() {
    const leafletCssUrl = 'https://unpkg.com/leaflet@latest/dist/leaflet.css';
    const leafletJsUrl = 'https://unpkg.com/leaflet@latest/dist/leaflet.js';

    if (!isLoaded(leafletCssUrl)) {
      injectStyleUrl(leafletCssUrl);
      injectScriptUrl(leafletJsUrl, () => {
        this.initMap();
      })
    } else {
      this.initMap();
    }
  },
  beforeDestroy() {
    if (this.map) {
      this.map.remove();
    }
  }
}