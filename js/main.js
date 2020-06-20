// Cloud-Constraints: Created by Jack Neylon, Phd, DABR (v0.1-2020.06.19)

var app = new Vue({
  el: '#app',
  components: { vuejsDatepicker },
  data: {
    message: 'Container for ConClouds',
    patient: '',
    mrn: '',
    rxdose: '',
    fractions: '',
    txdate: new Date(),
    rxdate: new Date(),
    physician: new Object(),
    physicist: new Object(),
    showHOME: true,
    showWAFER: 0,
    doctitle: 'Cloud-Constraints',
    docs:[
        {id:1,lastname:'Song',firstname:'River'},
        {id:2,lastname:'Oswald',firstname:'Clara'},
        {id:3,lastname:'Pond',firstname:'Amy'},
        {id:4,lastname:'Tyler',firstname:'Rose'}
      ],
    nerds: [
        {id:'1',lastname:'Tesla',firstname:'Nikola'},
        {id:'2',lastname:'Curie',firstname:'Marie'},
        {id:'3',lastname:'Heisenberg',firstname:'Werner'},
        {id:'4',lastname:'Bohr',firstname:'Niels'}
      ],
  },
  methods: {
    updateTitle: function(new_title) {
      this.doctitle = "ConCloud-" + this.mrn + "-" + new_title;
      document.title = this.doctitle;
    },
    resetTitle: function() {
      this.doctitle = "Cloud-Constraints";
      document.title = this.doctitle;
    },
    toggleShow: function(showSwitch) {
      if (showSwitch) {
        return(false)
      }
      else {
        return(true)
      }
    }
  }
})
