import {useRef,useState} from 'react'
import {request} from './api'

type Props={id:string;duration:number;captions:boolean;enableCaptions:boolean;resume:number;finished:()=>void;position:(seconds:number)=>void}
export function WatchPlayer({id,duration,captions,enableCaptions,resume,finished,position}:Props){
  const [attempt,setAttempt]=useState(0)
  const [failed,setFailed]=useState(false)
  const [ready,setReady]=useState(false)
  const initial=useRef(resume)
  const latest=useRef(resume)
  const lastSent=useRef(0)
  const [saveError,setSaveError]=useState(false)
  const base='/api/watch/media/'+encodeURIComponent(id)
  function progress(v:HTMLVideoElement,immediate:boolean){
    if(!v.readyState||v.ended||failed)return
    const seconds=Math.min(duration,v.currentTime)
    latest.current=seconds;position(seconds)
    if(seconds<=0 || (!immediate&&Date.now()-lastSent.current<10000))return
    lastSent.current=Date.now()
    void request('watch/bookmark','POST',{id,kind:'position',position:seconds}).then(()=>setSaveError(false)).catch(()=>setSaveError(true))
  }
  function fallback(){
    if(attempt>=2){setFailed(true);return}
    initial.current=latest.current;setReady(false);setAttempt(attempt+1)
  }
  return <>
    {attempt>0&&!ready&&!failed&&<p role="status">Preparing compatible playback…</p>}
    {failed?<p role="alert">This video could not be played. <button onClick={()=>{setFailed(false);setReady(false);setAttempt(0)}}>Try again</button></p>:
      <video key={attempt} controls preload="metadata" autoPlay={false} src={base+(attempt===0?'/file':attempt===1?'/compatible':'/compatible?encode=1')}
        onLoadedMetadata={e=>{const v=e.currentTarget;const start=initial.current;if(start>0&&start<v.duration)v.currentTime=start;setReady(true)}}
        onTimeUpdate={e=>progress(e.currentTarget,false)} onPause={e=>progress(e.currentTarget,true)}
        onEnded={finished} onError={fallback}>
        {captions&&<track kind="captions" label="Captions" default={enableCaptions} src={base+'/captions'}/>}
      </video>}
    {saveError&&<p role="status">Playback position could not be saved.</p>}
  </>
}
