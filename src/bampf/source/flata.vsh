#version 330

layout(location=0) in vec4 in_v;  // vertex coordinates

uniform mat4  mvpm;  // projection * modelView
uniform vec3  kd;    // diffuse colour
uniform float alpha; // transparency
out     vec4  v_c;   // vertex colour

void main() {
   gl_Position = mvpm * in_v;
   v_c = vec4(kd, alpha);
}
