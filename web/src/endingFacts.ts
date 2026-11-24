// Source URLs are retained for parent review. The child viewer reads only bundled data.
const collections=[
  {topic:'Space',source:'NASA Science',url:'https://science.nasa.gov/solar-system/solar-system-facts/',facts:[
    'The Sun is the star at the center of our solar system.',
    'Eight planets travel around the Sun.',
    'Mercury has no moons.',
    'Venus has no moons either.',
    'Pluto is a dwarf planet beyond Neptune.',
    'Our solar system is part of the Milky Way galaxy.',
    'The icy region beyond Neptune is called the Kuiper Belt.',
    'Some asteroids have moons of their own.',
  ]},
  {topic:'Animals',source:'Smithsonian National Museum of Natural History',url:'https://naturalhistory.si.edu/education/teaching-resources/life-science/what-insect',facts:[
    'A bee has six legs. Try counting three on each side.',
    'A spider has eight legs.',
    'An insect’s legs grow from its middle body section.',
    'A butterfly has two pairs of wings.',
    'A beetle keeps its flying wings under hard covers.',
    'A fly has tiny balancing organs shaped like clubs.',
    'An insect can sense its surroundings with two antennae.',
    'A flea is an insect without wings.',
  ]},
  {topic:'Animals',source:'NOAA',url:'https://www.noaa.gov/education/resource-collections/marine-life/marine-mammals',facts:[
    'Whales breathe air through lungs.',
    'A mother dolphin feeds her calf milk.',
    'Seals and sea lions are marine mammals.',
    'Manatees and dugongs belong to the same group of marine mammals.',
  ]},
  {topic:'Weather',source:'National Weather Service',url:'https://www.weather.gov/crp/weather_education',facts:[
    'Wind is moving air.',
    'A cloud can hold tiny drops of water.',
    'Cold clouds can hold ice crystals.',
    'Tiny water drops in a cloud can bump into each other and join.',
    'When water evaporates from your wet skin, it carries heat away.',
    'Rain can soak into the ground.',
    'Water can travel from a stream to a river and on to the ocean.',
    'Plants release water vapor into the air.',
  ]},
  {topic:'Plants',source:'US Forest Service',url:'https://www.fs.usda.gov/wildflowers/pollinators/documents/simpletruthbrochure.pdf',facts:[
    'A bee can carry pollen from one flower to another.',
    'Wind can move pollen between plants.',
    'Water carries pollen for some plants.',
    'Nectar gives pollinators energy.',
    'Pollen can provide protein for a pollinator.',
    'A flower’s scent can help attract its pollinators.',
    'Some animals visit flowers to collect nest materials.',
    'Pollination helps flowering plants make seeds.',
  ]},
  {topic:'Measurement',source:'National Institute of Standards and Technology',url:'https://www.nist.gov/system/files/documents/2025/12/23/2026_NIST_HB133_AppendixE-20251222.pdf',facts:[
    'A kilometer is a thousand meters.',
    'Ten millimeters make one centimeter.',
    'A meter is a hundred centimeters.',
    'A liter holds a thousand milliliters.',
  ]},
]
export const factTopics=[...new Set(collections.map(group=>group.topic))]
export function chooseFact(topic:string){
  const choices=collections.filter(group=>!topic||group.topic===topic).flatMap(group=>group.facts.map(text=>({text,source:group.source,url:group.url})))
  let previous=''
  try{previous=sessionStorage.getItem('familytv-last-fact')||''}catch{/* Storage is optional on TV. */}
  const pool=choices.filter(f=>f.text!==previous)
  const fact=pool[Math.floor(Math.random()*pool.length)]||choices[0]
  if(fact)try{sessionStorage.setItem('familytv-last-fact',fact.text)}catch{/* Offline sign-off still works. */}
  return fact
}
