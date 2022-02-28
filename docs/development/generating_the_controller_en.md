# Generating the dogu operator

This document contains the instruction performed to generate the controller. For this, `kubebuilder` was used.

## Install kubebuilder

For installation instructions see https://book.kubebuilder.io/quick-start.html#installation.

## Generate controller

The following command was executed in the root of the controller.

1. Generate the initial structure of the controller:

`kubebuilder init --domain cloudogu.com --repo github.com/cloudogu/k8s-dogu-operator`

2. Generate the api and CRD for the Dogus

`kubebuilder create api --group k8s --version v1 --kind Dogu`