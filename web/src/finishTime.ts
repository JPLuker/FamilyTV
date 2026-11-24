// Resolve the wall time in this browser's displayed time zone. Inspect both sides
// of a transition so an ambiguous or missing local minute is visible to the parent.
export function finishInstants(value:string):Date[]{
  if(!/^\d{4}-\d\d-\d\dT\d\d:\d\d$/.test(value))return []
  const wall=Date.parse(value+'Z')
  const offsets=new Set([-1,0,1].map(day=>new Date(wall+day*86400000).getTimezoneOffset()))
  return [...offsets].map(offset=>new Date(wall+offset*60000)).filter(d=>{
    return new Date(d.getTime()-d.getTimezoneOffset()*60000).toISOString().slice(0,16)===value
  }).sort((a,b)=>a.getTime()-b.getTime())
}
